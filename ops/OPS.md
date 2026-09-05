# The ops corner

Four extra services ride along in `docker-compose.yml` so the whole operations story can
be practised locally before a real server exists. When one does, run the same compose
there and everything comes with you.

| Service | Where | What it is |
|---|---|---|
| Mailpit | http://localhost:8025 | Catches every letter the service sends (invitations, password resets, forwarded feedback). Nothing leaves the machine; open the inbox in the browser. When a real mailbox arrives, point `STRATA_MAIL_*` at it instead of `strata-mail:1025`. |
| Uptime Kuma | http://localhost:3001 | The watchman. First visit asks you to create an account; then add two monitors: `http://strata-api:5000/health` and `http://strata-web:80` (use these in-network names, not localhost). It keeps uptime history and can page you later. |
| Prometheus | http://localhost:9090 | Scrapes the service pulse from `/metrics` every 15 seconds. Check Status > Targets to see the scrape is healthy. |
| Grafana | http://localhost:3002 | Dashboards over Prometheus (datasource is provisioned automatically). Sign in with `STRATA_GRAFANA_USER` / `STRATA_GRAFANA_PASSWORD` from `.env`, default admin / admin123. |

Bring them up with:

```sh
docker compose up -d strata-mail strata-uptime strata-prometheus strata-grafana
```

A known honest limit: everything here lives and dies with this machine and Docker
Desktop. A watchman in the same house cannot report that the house burned down - move
this to a server the moment one exists.
