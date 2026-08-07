# FireReach API — Swagger / OpenAPI Documentation Design Spec

**Date:** 2026-08-07
**Status:** Approved, ready for implementation planning
**Scope:** `api/` repository only. The `app/` repository is not touched.

## Problem

The FireReach API exposes 15 routes and has no machine-readable description of any of them. `internal/infra/router/router.go` is the only source of truth, so anyone integrating — a frontend developer, a future mobile client, a partner — has to read Go source to learn the request and response shapes.

This project has already been bitten by undocumented contract drift: the `primary_phone` JSON tag had no test pinning it, and a rename would have compiled cleanly while silently breaking the mobile app.

## Decisions

| # | Decision | Rationale |
|---|---|---|
| 1 | Branch `feature/swagger-docs` from `feature/api-wiring` (`a242131`) | That branch carries `distance_meters` and `primary_phone`. Branching from `main` would generate a spec advertising a `distance` field that is never populated and omitting `primary_phone` entirely — actively misleading documentation. |
| 2 | Serve the UI only when `cfg.Environment != "production"` | Keeps the admin route surface, the auth scheme, and internal error strings out of public view. `router.New` already receives the config, so this is one conditional. |
| 3 | Document all 15 routes, admin included | The admin routes are the least discoverable and benefit most. `@Security BearerAuth` makes them exercisable from the UI. |
| 4 | Type the success responses (7 edits); leave the 46 error responses as `gin.H` | swaggo derives schemas from the annotation, not the handler body, and `gin.H{"error": x}` marshals identically to `ErrorResponse{Error: x}`. Typing all 52 sites would be 52 mechanical edits for no change in generated output. |
| 5 | Commit the generated `docs/` output | `go build` never needs the `swag` CLI — same reasoning as committing the seed SQL rather than requiring network access at build time. |

## Architecture

### Dependencies

Three additions to `go.mod`: `github.com/swaggo/swag`, `github.com/swaggo/gin-swagger`, `github.com/swaggo/files`. The `swag` CLI is invoked via `go run github.com/swaggo/swag/cmd/swag@latest` from a Makefile target, so no global install is required.

### Response types

`internal/adapter/dto/common.go` (new) declares four types. All four already describe shapes the API returns today; none changes a byte on the wire.

```go
type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type CreatedAdminResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
```

### Handler body changes — 7 edits, exhaustively

| File | Current | Becomes |
|---|---|---|
| `handler/station.go` | `gin.H{"message": "station updated"}` | `dto.MessageResponse{Message: "station updated"}` |
| `handler/station.go` | `gin.H{"message": "station deactivated"}` | `dto.MessageResponse{Message: "station deactivated"}` |
| `handler/submission.go` | `gin.H{"message": "submission created"}` | `dto.MessageResponse{Message: "submission created"}` |
| `handler/submission.go` | `gin.H{"message": "submission reviewed"}` | `dto.MessageResponse{Message: "submission reviewed"}` |
| `handler/auth.go` (Register) | `gin.H{"id": id, "email": req.Email}` | `dto.CreatedAdminResponse{ID: id, Email: req.Email}` |
| `handler/auth.go` (Setup) | `gin.H{"id": id, "email": req.Email}` | `dto.CreatedAdminResponse{ID: id, Email: req.Email}` |
| `handler/auth.go` | unexported `loginResponse` type and its use | `dto.LoginResponse` — the unexported type is deleted |

**Nothing else in any handler body changes.** The 46 `gin.H{"error": ...}` sites and the 6 in `router/middleware/` stay exactly as they are; they are described by `@Failure ... {object} dto.ErrorResponse` annotations.

Because every one of these seven is wire-identical, the existing test suite passing unchanged is the proof that no response shape moved. No test may be modified to accommodate this refactor — if a test needs changing, the refactor is wrong.

### Annotations

General API metadata goes above `main()` in `cmd/api/main.go`:

```go
// @title           FireReach API
// @version         1.0
// @description     Emergency fire-station lookup and reporting for Ghana.
// @BasePath        /v1

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                JWT from POST /v1/auth/login, as "Bearer <token>".
```

Each handler method gets a block declaring summary, tags, params, and every response it can actually return. The tag set is `stations`, `submissions`, `content`, `ai`, `auth`, `admin`.

Routes to document, all 15:

**Public (8)** — `GET /stations`, `GET /stations/{id}`, `POST /submissions`, `POST /ai/ask`, `GET /content`, `GET /content/{id}`, `POST /auth/login`, `POST /auth/setup`

**Admin (7, all `@Security BearerAuth`)** — `GET /admin/submissions`, `PATCH /admin/submissions/{id}`, `POST /admin/users`, `GET /admin/users`, `POST /admin/stations`, `PATCH /admin/stations/{id}`, `DELETE /admin/stations/{id}`

Declared failure responses must match what the handler actually returns. `GET /stations` returns 400 for a missing, unparseable, NaN, or out-of-range coordinate and 500 on a repository error — both get documented. The two rate-limited routes (`POST /submissions`, `POST /ai/ask`) additionally declare 429, and every admin route declares 401.

### Generation and serving

`swag init -g cmd/api/main.go` writes `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml` under `api/`. A `make swagger` target wraps it, added to `.PHONY`.

`router.go` gains a blank import of the generated `docs` package plus:

```go
if cfg.Environment != "production" {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

The route sits outside the `/v1` group, so the UI lives at `/swagger/index.html`.

## Testing

Two tests in the existing `tests/` harness, which already builds a router via `newTestApp()`.

**1. Environment gating.** `/swagger/index.html` returns 200 when the environment is non-production and 404 when it is `production`.

`newTestApp` currently builds `&config.Config{JWTSecret: "test-jwt-secret"}`, leaving `Environment` as the zero value `""`, which already satisfies `!= "production"` — so the existing harness gets the route mounted with no change. To reach the production case, extract the body into `newTestAppWithEnv(env string)` and make `newTestApp()` call it with `""`. Every existing call site keeps working untouched.

**2. Drift detection.** Parse the committed `docs/swagger.json`, enumerate the routes registered on the Gin engine via `router.Routes()`, and assert every non-`/swagger` route appears in the spec's `paths` with its method.

The second test is the point of the exercise. Generated documentation rots silently — someone adds an endpoint, forgets the annotation, and the spec quietly lies. The test needs no CLI at run time, costs about thirty lines, and fails the moment a route goes undocumented. Gin's path params (`:id`) must be translated to OpenAPI's (`{id}`) when comparing.

## Out of scope

Changing any response shape, status code, or route. Typing the 46 error-response call sites. Documenting request or response examples beyond what the DTO types convey. CI wiring. Publishing the spec anywhere. Authentication changes. Anything in the `app/` repository.

## Notes

The superpowers planning artifacts for this work live under `api/docs/superpowers/`, while swag generates into `api/docs/` directly. They coexist as siblings and `swag init` does not remove unrelated files. If generation proves to interfere, relocate swag's output with `swag init -o docs/openapi` rather than moving the planning artifacts.

Earlier planning artifacts for the cross-repo app↔API work live in `app/docs/superpowers/`, because that work's primary deliverable was app-side. This spec is api-only and is therefore filed in the api repository.
