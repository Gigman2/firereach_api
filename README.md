# FireReach API

Go backend for FireReach: emergency fire-station lookup, community station
reporting, and fire-safety guidance for Ghana.

> **Station data is development seed data and is NOT verified for emergency
> use.** Contact numbers were archived from the GNFS site on 2022-08-09, are
> regional command lines rather than per-station direct lines, and may be
> stale. Confirm every number directly with the Ghana National Fire Service
> before any production or emergency-facing deployment.

## Stack

| Concern | Choice |
| --- | --- |
| Language | Go 1.25 |
| HTTP | Gin |
| Database | PostgreSQL 16 via pgx/v5 |
| Auth | JWT (HS256) + bcrypt |
| AI | Anthropic SDK for Go |
| Docs | swaggo / gin-swagger |
| Logging | zerolog |

## Quick start

The fastest path is Docker, which brings up Postgres and the API together:

```bash
cp .env.example .env      # then fill in CLAUDE_API_KEY and JWT_SECRET
make docker-up
make migrate-up
make seed                 # 57 stations, 165 contacts
make seed-content         # safety guides
```

The API listens on `http://localhost:9000`.

To run against a local Go toolchain instead, start Postgres however you
prefer, point `DATABASE_URL` at it, then:

```bash
make run       # go run ./cmd/api
make dev       # same, with nodemon watching *.go
```

### Environment

Loaded from `.env` in development via godotenv, and from the real environment
everywhere else. `DATABASE_URL` and `JWT_SECRET` are required and the process
exits without them.

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `DATABASE_URL` | yes | none | Postgres connection string |
| `JWT_SECRET` | yes | none | Signing key for admin tokens |
| `CLAUDE_API_KEY` | for AI | none | Without it, `POST /v1/ai/ask` fails |
| `ENVIRONMENT` | no | `production` | Gates the Swagger UI, see below |
| `PORT` | no | `9000` | |

### Migrations

Migrations run through [golang-migrate](https://github.com/golang-migrate/migrate),
which the Makefile expects at `$HOME/go/bin/migrate`:

```bash
make migrate-up
make migrate-down
```

## Architecture

Clean architecture, with dependencies pointing inward. `cmd/api/main.go` is
the only place that knows about every layer; it constructs the graph
explicitly rather than using a DI container, so the wiring reads top to bottom.

```
cmd/api/          entrypoint and dependency wiring
internal/
  domain/         entities, repository interfaces, business rules. No imports
                  from outer layers.
  usecase/        one type per operation (ListNearestStations, AskAI, ...),
                  depending only on domain interfaces
  adapter/
    handler/      Gin handlers, HTTP concerns only
    dto/          request and response shapes
  infra/
    postgres/     connection pool
    repo/         repository implementations (hand-written pgx)
    claude/       Anthropic gateway and the system prompt
    config/       environment loading
    router/       route table and middleware
migrations/       golang-migrate SQL, up and down
seeds/            dev_stations.sql, safety_content.sql
tests/            integration and end-to-end tests
```

A note on `sqlc.yaml`: sqlc is configured but not currently used. There is no
`internal/infra/postgres/queries/` directory and no generated code. The
repositories under `internal/infra/repo/` are hand-written against pgx.

## API surface

Base path `/v1`. Public routes need no credentials.

### Public

| Method | Path | Notes |
| --- | --- | --- |
| `GET` | `/v1/stations` | Nearest active stations, ranked by great-circle distance from the supplied coordinates |
| `GET` | `/v1/stations/:id` | Single station |
| `GET` | `/v1/content` | Safety guides |
| `GET` | `/v1/content/:id` | Single guide |
| `POST` | `/v1/submissions` | Community station report. Rate limited. |
| `POST` | `/v1/ai/ask` | Fire-safety question. Rate limited. |
| `POST` | `/v1/auth/login` | Returns a 24h JWT |
| `POST` | `/v1/auth/setup` | Bootstraps the first admin. Returns 403 once any admin exists. |

### Admin

All admin routes require `Authorization: Bearer <token>`.

| Method | Path | Notes |
| --- | --- | --- |
| `GET` | `/v1/admin/submissions` | Pending community reports |
| `PATCH` | `/v1/admin/submissions/:id` | Approve or reject, promoting an approved report into a station |
| `POST` | `/v1/admin/users` | Create another admin |
| `GET` | `/v1/admin/users` | List admins |
| `POST` | `/v1/admin/stations` | Create |
| `PATCH` | `/v1/admin/stations/:id` | Update |
| `DELETE` | `/v1/admin/stations/:id` | Deactivate (soft delete) |

### Request validation

Every `:id` is a UUID. Anything else answers `400 {"error":"invalid id"}` rather
than reaching Postgres, which rejects a malformed id with an error the
repositories would report as a server fault.

| Field | Rule |
| --- | --- |
| `type` (submission) | One of `wrong_phone`, `wrong_location`, `missing`, `closed`. These strings are the contract with the app. |
| `suggested_value` | Required, not blank, at most 1000 characters |
| `note`, `admin_note` | At most 1000 characters |
| `device_hash` | 1 to 64 letters, digits, `_` or `-`. A rate-limit bucket key, not a credential. |
| `station_id` | A UUID, or absent when the station is not on file. An id matching no station answers 400. |
| `question` (AI) | Required, at most 1000 characters |
| `topic` (AI) | At most 100 characters |
| `history` (AI) | At most 50 turns, each at most 2000 characters |
| `lat`, `lng` | Between -90 and 90, and -180 and 180. 0 is a real value: the prime meridian runs through Tema. |
| `password` | At most 72 bytes, which is bcrypt's own limit |

Text limits count characters, not bytes, so a report written in Twi gets the
same room as one written in English.

### First admin

`/v1/auth/setup` is deliberately unauthenticated and self-closing: it creates
an admin only while the `admin_users` table is empty, then refuses forever.

```bash
curl -X POST localhost:9000/v1/auth/setup \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"..."}'
```

## Rate limiting

`POST /v1/submissions` and `POST /v1/ai/ask` are limited to 10 requests per
minute with a burst of 5, keyed on the `X-Device-Hash` header and falling back
to the client IP. Limiter state is in-process, so it resets on restart and is
per-instance rather than global.

## AI answers

`POST /v1/ai/ask` returns a structured answer, not free prose. The model
picks a `kind` and the client renders the matching component:

`text`, `emergency`, `steps`, `emergency_number`, `warning`, `out_of_scope`

Two properties matter more than the shape:

1. **No kind carries a phone number.** The app owns every number and every
   dial control, which is what stops a hallucinated digit from reaching a
   call button.
2. **Unknown or malformed payloads degrade to `text`.** A client that does
   not recognise a kind renders the body alone rather than failing.

The system prompt lives in `internal/infra/claude/gateway.go` alongside
`EmergencyPhone`, so Ghana's emergency number is defined in exactly one place
and the adversarial suite reuses that constant.

## Swagger

Interactive docs are served at `/swagger/index.html`, but only when
`ENVIRONMENT` is one of `development`, `staging`, or `test`. The check is an
allow-list rather than a "not production" test on purpose, so an unrecognised
or misspelled value fails closed. A missing docs UI in development gets
noticed in minutes; a docs UI exposed in production does not.

Regenerate the spec after changing annotations:

```bash
make swagger      # go tool swag init -g cmd/api/main.go
```

## Testing

```bash
make test         # go test ./...
```

24 test files covering domain rules, use cases with mocks, content hashing and
visibility, and end-to-end HTTP flows.

The database-backed tests under `tests/` **skip when `DATABASE_URL` is unset**,
so a bare `go test ./...` can report all-green without having run them. `make
test` does set it, because the Makefile does `include .env`, which means the
Make target needs Postgres actually up (`make docker-up`). If you want the
integration tests to run, use `make test`; if you only want the pure units,
`go test ./...` with no `.env` in the environment is the faster loop.

The adversarial suite for the AI gateway is excluded by a build tag because it
hits the live Anthropic API, costs money, and is non-deterministic in ways
that would make a red build meaningless:

```bash
CLAUDE_API_KEY=... go test -tags=adversarial ./internal/infra/claude/ -v
```

Run it before a release. Every escape found while hardening it became a
permanent case in the tables there, so add cases rather than loosening the
assertions.

## Make targets

| Target | Does |
| --- | --- |
| `run` | Start the API |
| `dev` | Start with file watching |
| `build` | Binary to `bin/firereach-api` |
| `test` | `go test ./...` |
| `migrate-up` / `migrate-down` | Apply or roll back migrations |
| `seed` | Load `seeds/dev_stations.sql` |
| `seed-content` | Load `seeds/safety_content.sql` |
| `docker-up` / `docker-down` | Compose stack |
| `swagger` | Regenerate the OpenAPI spec |
| `tidy` | `go mod tidy` |

## Relationship to the app repo

`seeds/safety_content.sql` is **generated**, not authored here. Its source of
truth is `src/data/safety-content.bundled.json` in the FireReach app repo, and
it is regenerated by `npm run build:content-seed` from that repo. Edit the
guide text there, not in this SQL file. Each item carries a `content_hash` so
an edit after a clinical review demotes that item's review state rather than
silently keeping the approval.

## License

FireReach API is free software, licensed under the **GNU Affero General Public
License v3.0 or later**. See [`LICENSE`](LICENSE).

    Copyright (C) 2026 Eric Abbey

    This program is free software: you can redistribute it and/or modify it
    under the terms of the GNU Affero General Public License as published by
    the Free Software Foundation, either version 3 of the License, or (at your
    option) any later version.

    This program is distributed in the hope that it will be useful, but
    WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
    or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public
    License for more details.

Commercial use is permitted. The condition is reciprocity: if you modify
FireReach and run it as a network service, AGPL section 13 requires you to
offer your users the corresponding source of your modified version.

### Deployment note

Section 13 attaches to modified versions offered over a network. If you deploy
a fork, publish its source and link to it from the service so users interacting
with the API can reach it.

### Station data is licensed separately

`seeds/dev_stations.sql` is **not** under the AGPL. It derives from
OpenStreetMap and is offered under the **Open Database License (ODbL) v1.0**.
See [`LICENSE-DATA`](LICENSE-DATA).

    Contains information from OpenStreetMap, which is made available under
    the Open Database License (ODbL) v1.0.
    (c) OpenStreetMap contributors - https://www.openstreetmap.org/copyright

This split is not optional. ODbL share-alike passes to derived databases, so
that file has to stay ODbL regardless of how the surrounding code is licensed.
The telephone numbers in it come from the GNFS published contact page rather
than OpenStreetMap, and `response_rate` is FireReach's own derivation. Details
are in `LICENSE-DATA`.
