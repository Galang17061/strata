# Strata

Strata is a reliability analysis service built around the reliability block diagram method. An engineer models a system as a tree of subsystems and components, draws how the blocks are wired (series, parallel, mixed, k-out-of-n), records every failure a component has had, and Strata works out, from the components upward, how likely each block and the whole system is to still be working after a given number of running hours. It also draws that chance as a curve over time.

This repository is the backend. It serves a REST API for the Next.js frontend, keeps its data in PostgreSQL, and is shipped as a single static binary.

## What it does

- Sign in with a username and password and carry a bearer token on every request; manage users, roles, and per-module access rights.
- Keep master data on record: manufacturers (with their logo), components (with Excel import, export, and a template), and projects.
- Lay out a whole system tree in one call: system, subsystems on up to three levels, components, and the virtual in/out nodes the canvas needs.
- Wire blocks on the canvas at component level or hierarchy level; every saved wiring regenerates the formula of the block and of every block above it.
- Log failure events per component and derive failure rate, mean time between failures, and either a Weibull (shape, scale) or an exponential model from them.
- Score reliability bottom-up: component R(t), then each hierarchy level, then the system, with series, parallel, and k-out-of-n redundancy, and keep a history of every calculation.
- Produce reliability plots as time series for components, hierarchies, and systems.

## Layout

```
cmd/strata            entrypoint and the migrate subcommand
internal/config       STRATA_* environment settings
internal/database     connection, database creation, schema migration
internal/auth         token issuing and parsing, password cipher, request guard
internal/web          router, response envelopes, paging, static files, health
internal/domain       entities, request and response shapes, decimal number type
internal/account      auth, users, roles, user access, settings
internal/master       manufacturers, components, projects
internal/rbd          systems, hierarchies, component properties, drawings, failures, distributions, totals, plots
internal/reliability  distributions, formula from topology, formula evaluation, regression, decimal arithmetic
migrations            SQL that creates the database and the schema
docs                  OpenAPI description served at /swagger
```

## Configuration

Every setting is an environment variable with the `STRATA_` prefix. Copy `.env.example` to `.env` and fill in real values; `.env` is never committed.

| Variable | Meaning |
|---|---|
| `STRATA_PORT` | Port the HTTP server listens on (default `5000`) |
| `STRATA_DB_CONNECTION` | PostgreSQL URL, for example `postgres://postgres:secret@localhost:5432/strata?sslmode=disable` |
| `STRATA_DB_SA_PASSWORD` | Password given to the PostgreSQL container started by docker compose |
| `STRATA_JWT_KEY` | HMAC secret for bearer tokens, at least 64 characters |
| `STRATA_JWT_ISSUER`, `STRATA_JWT_AUDIENCE` | Claims written into every token (defaults `issuer`, `audience`) |
| `STRATA_JWT_EXPIRY_MINUTES` | Token lifetime (default `10080`, seven days) |
| `STRATA_PASSWORD_KEY` | Key of the symmetric cipher used for stored passwords |
| `STRATA_UPLOAD_DIR` | Directory for uploaded files, served under `/files` (default `./upload`) |
| `STRATA_MIGRATIONS_DIR` | Directory holding the SQL migrations (default `./migrations`) |

## Running locally

Requirements: Go 1.25 or newer and a reachable PostgreSQL.

```
cp .env.example .env
go build -o bin/strata ./cmd/strata
./bin/strata migrate
./bin/strata
```

`strata migrate` creates the `strata` database when the login is allowed to, then applies `migrations/0001_schema.sql`, which is idempotent and can be run again at any time. The server answers on `http://localhost:5000`; `GET /health` reports liveness and `/swagger/index.html` opens the interactive API description.

## Running with Docker

The compose file starts two services on the `strata-network` network: `strata-db` (PostgreSQL 17 on port 5432, data kept in the `strata-pg-data` volume) and `strata-api` (port 5000, uploads kept in the `strata-upload` volume). The API container runs the migration and then serves.

```
cp .env.example .env
docker compose up -d --build
docker compose ps
```

`.env` feeds both services: `STRATA_DB_SA_PASSWORD` sets the PostgreSQL password and `STRATA_DB_CONNECTION` must point at `strata-db:5432` with the same password. When the API runs outside Docker against that database, use `localhost:5432` in the URL instead.

## Tests

```
go build ./... && go vet ./... && go test ./...
```

The suite covers the decimal arithmetic, the distributions, the formula built from a drawing, the evaluation of that formula, the paging and routing helpers, and thirty-nine drawn topologies from single blocks to meshed multi-stage systems, each pinned to the formula text and the reliability figure the previous service produced for it.

## Moving data from the previous service

The schema is identical to the one the previous service used: same table names, same column names and spelling, same types and decimal precision. Data can be copied table by table with `INSERT ... SELECT` in this order: Roles, Users, UserRole, UserAccess, MasterManufacturer, MasterComponent, MasterProject, RbdSystemDrawing, Hierarchy, SystemComponentProperties, SystemComponentDrawing, FailureEventHistory, WeibullParameter, ExponentialParameter, ReliabilityPlotComponent, ReliabilityHistory, OptimizationRun, OptimizationResult. Stored passwords keep working because the cipher and its key are the same.

## What stayed the same, and what changed

Kept on purpose, so the frontend only needs a new base URL:

- Every route, method, query and body shape, status code, and header.
- The response envelope (`status`, `statusCode`, `message`, `data`, `meta`), the validation problem document, and the exception body.
- Case-insensitive paths and case-insensitive JSON keys.
- All reliability figures to eight decimal places, computed with decimal arithmetic that mirrors the previous engine, including the way the formula for a drawing is written out.
- The database schema and the password cipher.

Changed underneath:

- One static Go binary instead of a managed runtime; configuration through environment variables only, no secrets in the repository.
- Explicit SQL through a small store layer instead of an ORM.
- Every capability covered by tests that run in well under a second.
- A `PUT /api/User/ChangePasswordAdmin?UserId=` route that lets an administrator set a user's password without the old one, because the frontend already calls it.

Left as it was, to be revisited once parity is confirmed in production: open CORS, seven-day tokens without refresh, and authorization that checks only that the caller is signed in.
