# The load rehearsal

Everything needed to ask "how many people can lean on this at once" without touching a
byte of real data. The rig plays five kinds of user against a **twin of the service**
(`strata-api-test`, port 5001) that stands on a **throwaway copy of the database**
(`stratatest`) restored from the latest backup.

## One-time per rehearsal

```sh
sh ops/load/seed-testdb.sh
docker compose --profile load up -d strata-api-test
```

The seed script takes a fresh backup, restores it as `stratatest`, and writes
`STRATA_TEST_DB_CONNECTION` into `.env` on first run. Re-run it whenever you want the
twin reset to today's data.

## Running

```sh
docker compose --profile load run --rm -e MODE=smoke strata-load
docker compose --profile load run --rm -e MODE=load strata-load
docker compose --profile load run --rm -e MODE=stress strata-load
docker compose --profile load run --rm -e MODE=soak -e SOAK=2h strata-load
```

- **smoke** — 5 users for a minute; proves the journeys themselves are healthy.
- **load** — ramp to 100 users and hold; the "ordinary day" test against the SLOs.
- **stress** — climb 50 → 400 users; where the thresholds start failing is the capacity
  of this machine.
- **soak** — 60 users for a long stretch; watch `strata_heap_alloc_bytes` in Grafana
  for slow leaks.

## The cast

55% readers (dashboard, projects, tree, totals, plot), 25% engineers (component sheet,
suggestion, a full recalculate), 10% heavy users (a genetic search - the studio gate
turns extras away politely and the script counts them as `ga_turned_away`, not as
errors), 5% admins (users, audit), 5% wanderers (alerts, versions, the odd feedback
note). Every action is followed by one to four seconds of human hesitation.

## Reading the results

k6 prints latency percentiles per kind (read / recalc / ga) and pass/fail against the
thresholds baked into `journeys.js`. Watch the same run from the server's side in
Grafana (http://localhost:3002) via the `strata_*` metrics.

Honest caveat: on this laptop the attacker, the service and the database share one CPU,
so numbers here are a conservative floor and a regression baseline - the real 300-500
figure gets measured when the same rig points at a proper server.
