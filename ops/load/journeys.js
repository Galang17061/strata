import http from "k6/http";
import { check, sleep } from "k6";
import { Counter } from "k6/metrics";

const BASE = __ENV.TARGET || "http://strata-api-test:5000/api";
const MODE = __ENV.MODE || "smoke";
const SYSTEM = __ENV.SYSTEM || "00000010";
const COMPONENT = __ENV.COMPONENT || "SCP-00000348";

const gaTurnedAway = new Counter("ga_turned_away");

const profiles = {
  smoke: { scenarios: { mix: { executor: "constant-vus", vus: 5, duration: "60s" } } },
  load: {
    scenarios: {
      mix: {
        executor: "ramping-vus",
        startVUs: 0,
        stages: [
          { duration: "2m", target: 100 },
          { duration: "3m", target: 100 },
          { duration: "30s", target: 0 },
        ],
      },
    },
  },
  stress: {
    scenarios: {
      mix: {
        executor: "ramping-vus",
        startVUs: 0,
        stages: [
          { duration: "1m", target: 50 },
          { duration: "2m", target: 150 },
          { duration: "2m", target: 250 },
          { duration: "2m", target: 400 },
          { duration: "1m", target: 0 },
        ],
      },
    },
  },
  soak: { scenarios: { mix: { executor: "constant-vus", vus: 60, duration: __ENV.SOAK || "20m" } } },
};

export const options = Object.assign(
  {
    thresholds: {
      http_req_failed: ["rate<0.02"],
      "http_req_duration{kind:read}": ["p(95)<1000"],
      "http_req_duration{kind:recalc}": ["p(95)<8000"],
      "http_req_duration{kind:ga}": ["p(95)<60000"],
    },
  },
  profiles[MODE] || profiles.smoke,
);

let token = null;

function jsonHeaders() {
  return { "Content-Type": "application/json", Authorization: "Bearer " + token };
}

function ensureLogin() {
  if (token) return;
  const res = http.post(
    BASE + "/Auth/Login",
    JSON.stringify({ UserName: "admin", Password: "admin123" }),
    { headers: { "Content-Type": "application/json" }, tags: { kind: "login" } },
  );
  check(res, { "login ok": (r) => r.status === 200 });
  token = res.json("data.token");
}

function read(path, name) {
  const res = http.get(BASE + path, { headers: jsonHeaders(), tags: { kind: "read", name: name } });
  check(res, { [name + " ok"]: (r) => r.status === 200 });
}

function pause() {
  sleep(1 + Math.random() * 3);
}

function viewerJourney() {
  read("/MasterProject/recent", "recent");
  pause();
  read("/MasterProject?page=1&pageSize=12", "projects");
  pause();
  read("/MasterSystem/getRbdTreeView/" + SYSTEM, "tree");
  pause();
  read("/ReliabilityTotal/rbdSystem/" + SYSTEM + "/reliability-total", "totals");
  pause();
  read("/ReliabilityTotal/reliabilityPlot?rbdSystemId=" + SYSTEM, "plot");
}

function engineerJourney() {
  read("/MasterSystem/getRbdTreeView/" + SYSTEM, "tree");
  pause();
  read("/SystemComponentProperties/" + COMPONENT, "component");
  pause();
  read("/SystemComponentProperties/" + COMPONENT + "/suggested-parameters", "suggestion");
  pause();
  const res = http.post(
    BASE + "/ReliabilityTotal/system/" + SYSTEM + "/recalculate",
    JSON.stringify({ RunningHours: 1000 }),
    { headers: jsonHeaders(), tags: { kind: "recalc", name: "recalculate" }, timeout: "60s" },
  );
  check(res, { "recalculate ok": (r) => r.status === 200 });
}

function heavyJourney() {
  read("/MasterSystem/getRbdTreeView/" + SYSTEM, "tree");
  pause();
  const res = http.post(
    BASE + "/Optimization/preview",
    JSON.stringify({ RbdSystemId: SYSTEM, Mode: 1, PopulationSize: 100, MaxGenerations: 100 }),
    { headers: jsonHeaders(), tags: { kind: "ga", name: "ga-preview" }, timeout: "120s" },
  );
  const busy = res.status === 400 && String(res.body).indexOf("studio is full") >= 0;
  if (busy) gaTurnedAway.add(1);
  check(res, { "ga ok or politely busy": (r) => r.status === 200 || busy });
}

function adminJourney() {
  read("/User?page=1&pageSize=12", "users");
  pause();
  read("/Audit?page=1&pageSize=12", "audit");
}

function wandererJourney() {
  read("/Notification", "alerts");
  pause();
  read("/Snapshot/system/" + SYSTEM, "versions");
  pause();
  if (Math.random() < 0.3) {
    const res = http.post(
      BASE + "/Feedback",
      JSON.stringify({ Category: "question", Message: "Load rehearsal note " + Date.now() }),
      { headers: jsonHeaders(), tags: { kind: "write", name: "feedback" } },
    );
    check(res, { "feedback ok": (r) => r.status === 200 });
  }
}

export default function () {
  ensureLogin();
  const roll = Math.random();
  if (roll < 0.55) viewerJourney();
  else if (roll < 0.8) engineerJourney();
  else if (roll < 0.9) heavyJourney();
  else if (roll < 0.95) adminJourney();
  else wandererJourney();
  pause();
}
