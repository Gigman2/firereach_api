# FireReach API Swagger Documentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate and serve interactive OpenAPI documentation for all 15 FireReach API routes, reachable at `/swagger/index.html` outside production.

**Architecture:** swaggo derives an OpenAPI spec from structured comments above each handler. Response schemas come from named Go types, so the four untyped success responses gain real types first; the 46 `gin.H` error responses stay untouched and are described by annotation, since `gin.H{"error": x}` and `ErrorResponse{Error: x}` marshal identically. The generated spec is committed so `go build` never needs the CLI.

**Tech Stack:** Go 1.25, Gin, `swaggo/swag` + `swaggo/gin-swagger` + `swaggo/files`.

## Global Constraints

- **Repository:** `api/` only, at `/Users/ericabbey/Desktop/Projects/firereach/api`. Never touch the sibling `app/` repository.
- **Branch:** `feature/swagger-docs`, which sits on top of `feature/api-wiring`. Commit there. Never commit to `main`, never switch branches.
- **Spec:** `docs/superpowers/specs/2026-08-07-swagger-design.md`. Read it before starting.
- **No response shape, status code, or route may change.** Every edit in Task 1 is wire-identical by construction.
- **No existing test may be modified to accommodate a change.** The existing suite passing unmodified is the proof that nothing moved. If a test needs editing, the change is wrong — stop and report.
- **The 46 `gin.H{"error": ...}` call sites in handlers and the 6 in `internal/infra/router/middleware/` stay exactly as they are.** They are documented by annotation, not rewritten.
- **`@Router` paths are relative to the `/v1` base path** — write `/stations`, never `/v1/stations`.
- Verification is `go build ./... && go vet ./... && go test ./...`.
- An untracked `cmd/.DS_Store` exists. Ignore it; never commit or delete it.

## File structure

| File | Responsibility |
|---|---|
| `internal/adapter/dto/common.go` (new) | Generic response envelopes: `ErrorResponse`, `MessageResponse` |
| `internal/adapter/dto/auth.go` (new) | Auth request/response types moved out of the handler package |
| `internal/adapter/handler/*.go` | Route annotations; four success responses retyped |
| `cmd/api/main.go` | General API metadata and the `BearerAuth` security definition |
| `internal/infra/router/router.go` | Environment-gated `/swagger/*any` route |
| `docs/` (generated, committed) | `docs.go`, `swagger.json`, `swagger.yaml` |
| `Makefile` | `swagger` target |
| `tests/swagger_test.go` (new) | Gating test and route-drift test |

---

## Task 1: Typed request and response DTOs

**Files:**
- Create: `internal/adapter/dto/common.go`
- Create: `internal/adapter/dto/auth.go`
- Modify: `internal/adapter/handler/auth.go`, `internal/adapter/handler/station.go`, `internal/adapter/handler/submission.go`
- Test: `internal/adapter/dto/common_test.go` (create)

**Interfaces:**
- Consumes: nothing (first task)
- Produces: `dto.ErrorResponse{Error string}`, `dto.MessageResponse{Message string}`, `dto.CreatedAdminResponse{ID, Email string}`, `dto.LoginResponse{Token string}`, `dto.LoginRequest{Email, Password string}`, `dto.RegisterRequest{Email, Password string}` — all referenced by name in later tasks' annotations

- [ ] **Step 1: Write the failing test**

Create `internal/adapter/dto/common_test.go`. These assertions pin the JSON shapes the API already emits, so a later rename cannot silently change the wire format:

```go
package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/firereach/api/internal/adapter/dto"
)

func TestErrorResponse_MarshalsToErrorKey(t *testing.T) {
	blob, err := json.Marshal(dto.ErrorResponse{Error: "station not found"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got, want := string(blob), `{"error":"station not found"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestMessageResponse_MarshalsToMessageKey(t *testing.T) {
	blob, err := json.Marshal(dto.MessageResponse{Message: "station updated"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got, want := string(blob), `{"message":"station updated"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestCreatedAdminResponse_MarshalsToIDAndEmail(t *testing.T) {
	blob, err := json.Marshal(dto.CreatedAdminResponse{
		ID:    "11111111-1111-1111-1111-111111111111",
		Email: "admin@firereach.gh",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	want := `{"id":"11111111-1111-1111-1111-111111111111","email":"admin@firereach.gh"}`
	if got := string(blob); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLoginResponse_MarshalsToTokenKey(t *testing.T) {
	blob, err := json.Marshal(dto.LoginResponse{Token: "abc.def.ghi"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got, want := string(blob), `{"token":"abc.def.ghi"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLoginRequest_UnmarshalsEmailAndPassword(t *testing.T) {
	var req dto.LoginRequest
	if err := json.Unmarshal([]byte(`{"email":"a@b.c","password":"secret123"}`), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Email != "a@b.c" || req.Password != "secret123" {
		t.Errorf("got %+v", req)
	}
}

func TestRegisterRequest_UnmarshalsEmailAndPassword(t *testing.T) {
	var req dto.RegisterRequest
	if err := json.Unmarshal([]byte(`{"email":"a@b.c","password":"secret123"}`), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Email != "a@b.c" || req.Password != "secret123" {
		t.Errorf("got %+v", req)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/adapter/dto/ -run 'TestErrorResponse|TestMessageResponse|TestCreatedAdmin|TestLogin|TestRegister' -v
```

Expected: FAIL — compile errors, `undefined: dto.ErrorResponse` and the other five types.

- [ ] **Step 3: Create the generic envelopes**

Create `internal/adapter/dto/common.go`:

```go
package dto

// ErrorResponse is the shape every error path returns. Handlers emit this as a
// gin.H literal; the two marshal identically, and this type is what the OpenAPI
// annotations reference.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse acknowledges a successful mutation that has no body to return.
type MessageResponse struct {
	Message string `json:"message"`
}
```

- [ ] **Step 4: Create the auth types**

Create `internal/adapter/dto/auth.go`. These move out of `internal/adapter/handler/auth.go`, where they were unexported and would have appeared in the spec as `handler.loginRequest`. Every other request type in this API already lives in `dto`:

```go
package dto

// LoginRequest authenticates an existing admin.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest creates an admin account, used by both /auth/setup and
// /admin/users.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse carries the signed JWT.
type LoginResponse struct {
	Token string `json:"token"`
}

// CreatedAdminResponse describes a newly created admin account.
type CreatedAdminResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./internal/adapter/dto/ -v
```

Expected: PASS, including the pre-existing `station_test.go` cases.

- [ ] **Step 6: Swap the nine handler call sites**

In `internal/adapter/handler/auth.go`:
1. Delete the `registerRequest`, `loginRequest`, and `loginResponse` type declarations.
2. In `Login`: `var req loginRequest` → `var req dto.LoginRequest`; `c.JSON(http.StatusOK, loginResponse{Token: tokenString})` → `c.JSON(http.StatusOK, dto.LoginResponse{Token: tokenString})`.
3. In `Register`: `var req registerRequest` → `var req dto.RegisterRequest`; `c.JSON(http.StatusCreated, gin.H{"id": id, "email": req.Email})` → `c.JSON(http.StatusCreated, dto.CreatedAdminResponse{ID: id, Email: req.Email})`.
4. In `Setup`: the same two changes as `Register`.

In `internal/adapter/handler/station.go`:
5. `c.JSON(http.StatusOK, gin.H{"message": "station updated"})` → `c.JSON(http.StatusOK, dto.MessageResponse{Message: "station updated"})`
6. `c.JSON(http.StatusOK, gin.H{"message": "station deactivated"})` → `c.JSON(http.StatusOK, dto.MessageResponse{Message: "station deactivated"})`

In `internal/adapter/handler/submission.go`:
7. `c.JSON(http.StatusCreated, gin.H{"message": "submission created"})` → `c.JSON(http.StatusCreated, dto.MessageResponse{Message: "submission created"})`
8. `c.JSON(http.StatusOK, gin.H{"message": "submission reviewed"})` → `c.JSON(http.StatusOK, dto.MessageResponse{Message: "submission reviewed"})`

`submission.go` already imports `dto`; confirm `auth.go` and `station.go` do too (both already reference `dto`, so no import changes are expected).

**Change nothing else.** Every `gin.H{"error": ...}` stays.

- [ ] **Step 7: Run the full suite**

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: everything passes with **no test file modified**. If any existing test now fails, a response shape moved — revert and report rather than editing the test.

- [ ] **Step 8: Commit**

```bash
git add internal/adapter/dto/common.go internal/adapter/dto/auth.go internal/adapter/dto/common_test.go internal/adapter/handler/auth.go internal/adapter/handler/station.go internal/adapter/handler/submission.go
git commit -m "refactor(dto): give success responses named types for OpenAPI schemas"
```

---

## Task 2: swaggo wiring, generation, and the gated route

**Files:**
- Modify: `cmd/api/main.go`, `internal/infra/router/router.go`, `Makefile`, `go.mod`, `go.sum`
- Create (generated): `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`
- Test: `tests/swagger_test.go` (create), `tests/e2e_test.go` (extract a constructor)

**Interfaces:**
- Consumes: `dto.ErrorResponse` and friends from Task 1
- Produces: `make swagger`; a `/swagger/*any` route mounted only when `cfg.Environment != "production"`; `newTestAppWithEnv(env string) *testApp` in the tests package

- [ ] **Step 1: Add the dependencies**

The first three are libraries the built binary links against. The fourth registers the generator as a tracked *tool* dependency — Go 1.24 added the `tool` directive precisely so a code generator's version is pinned in `go.mod` rather than floating on `@latest`:

```bash
go get github.com/swaggo/swag
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files
go get -tool github.com/swaggo/swag/cmd/swag
```

Confirm `go.mod` gained a `tool github.com/swaggo/swag/cmd/swag` directive. This requires network access once; afterwards the version is fixed and every regeneration is reproducible.

- [ ] **Step 2: Add the general API annotations**

In `cmd/api/main.go`, immediately above `func main()`:

```go
// @title           FireReach API
// @version         1.0
// @description     Emergency fire-station lookup and community reporting for Ghana.
// @description     Station data is development seed data and is NOT verified for emergency use.
// @BasePath        /v1

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                JWT issued by POST /v1/auth/login. Send as "Bearer <token>".
func main() {
```

- [ ] **Step 3: Add the Makefile target**

Append the `swagger` target and extend `.PHONY`:

```makefile
.PHONY: run dev build test migrate-up migrate-down generate docker-up docker-down seed swagger

swagger:
	go tool swag init -g cmd/api/main.go
```

Recipe lines must start with a real TAB. `go tool swag` resolves through the `tool` directive added in Step 1, so the generator version is pinned by `go.mod` and regeneration is reproducible — unlike `go run ...@latest`, which would silently change output whenever upstream releases.

If `go tool swag` reports an unknown tool, Step 1's `go get -tool` did not take effect; re-run it and check `go.mod` before continuing.

- [ ] **Step 4: Generate the spec**

```bash
make swagger
```

Expected: creates `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`. It will report finding few or no endpoint annotations — correct at this stage, since Tasks 3 and 4 add them.

Confirm the generator did not disturb the planning artifacts:

```bash
ls docs/ docs/superpowers/specs/
```

Expected: the three generated files alongside the `superpowers/` directory, with the spec and plan still present. If `swag` removed them, stop and report — the fallback is `swag init -g cmd/api/main.go -o docs/openapi`.

- [ ] **Step 5: Mount the gated route**

In `internal/infra/router/router.go`, add to the imports:

```go
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/firereach/api/docs"
```

and immediately before `v1 := r.Group("/v1")`:

```go
	// Interactive API documentation. Withheld in production so the admin route
	// surface, the auth scheme, and internal error strings are not published.
	if cfg.Environment != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
```

- [ ] **Step 6: Make the test harness environment-aware**

In `tests/e2e_test.go`, rename `newTestApp` to `newTestAppWithEnv(env string) *testApp`, change its config line to carry the environment, and add a thin `newTestApp()` that delegates. Every existing call site keeps compiling untouched:

```go
func newTestApp() *testApp {
	return newTestAppWithEnv("")
}

func newTestAppWithEnv(env string) *testApp {
	// ... existing body unchanged, except the config line:
	cfg := &config.Config{JWTSecret: "test-jwt-secret", Environment: env}
	// ...
}
```

`Environment` defaults to `""`, which already satisfies `!= "production"`, so existing tests continue to exercise a router with the docs route mounted.

- [ ] **Step 7: Write the gating test**

Create `tests/swagger_test.go`:

```go
package tests

import (
	"net/http"
	"testing"
)

func TestSwaggerUI_ServedOutsideProduction(t *testing.T) {
	for _, env := range []string{"", "development", "staging"} {
		t.Run("env="+env, func(t *testing.T) {
			app := newTestAppWithEnv(env)
			w := app.request("GET", "/swagger/index.html", nil)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200 for environment %q, got %d", env, w.Code)
			}
		})
	}
}

func TestSwaggerUI_WithheldInProduction(t *testing.T) {
	app := newTestAppWithEnv("production")
	w := app.request("GET", "/swagger/index.html", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 in production, got %d", w.Code)
	}
}
```

- [ ] **Step 8: Run the tests**

```bash
go test ./tests/ -run TestSwaggerUI -v
```

Expected: PASS, all four subtests.

- [ ] **Step 9: Run the full suite**

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: all packages pass.

- [ ] **Step 10: Commit**

```bash
git add go.mod go.sum Makefile cmd/api/main.go internal/infra/router/router.go docs/docs.go docs/swagger.json docs/swagger.yaml tests/e2e_test.go tests/swagger_test.go
git commit -m "feat(docs): serve swagger UI outside production"
```

---

## Task 3: Annotate the eight public routes

**Files:**
- Modify: `internal/adapter/handler/station.go`, `internal/adapter/handler/content.go`, `internal/adapter/handler/ai.go`, `internal/adapter/handler/submission.go`, `internal/adapter/handler/auth.go`
- Regenerate: `docs/`

**Interfaces:**
- Consumes: the `dto` types from Task 1; `make swagger` from Task 2
- Produces: eight documented paths in `docs/swagger.json`

Add each block directly above its handler method. Add nothing else — no code changes in this task.

- [ ] **Step 1: Annotate `StationHandler.ListNearest`**

```go
// ListNearest godoc
// @Summary      List nearest fire stations
// @Description  Returns active stations ordered by distance from the supplied coordinates, nearest first.
// @Tags         stations
// @Produce      json
// @Param        lat    query     number  true   "Latitude, -90 to 90"
// @Param        lng    query     number  true   "Longitude, -180 to 180"
// @Param        limit  query     integer false  "Maximum results (default 3)"
// @Success      200    {array}   dto.StationResponse
// @Failure      400    {object}  dto.ErrorResponse  "Missing, unparseable, NaN, or out-of-range coordinate"
// @Failure      500    {object}  dto.ErrorResponse
// @Router       /stations [get]
```

- [ ] **Step 2: Annotate `StationHandler.GetByID`**

```go
// GetByID godoc
// @Summary      Get a station by ID
// @Description  Returns one station. distance_meters is 0 here, since no reference coordinate is supplied.
// @Tags         stations
// @Produce      json
// @Param        id   path      string  true  "Station UUID"
// @Success      200  {object}  dto.StationResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /stations/{id} [get]
```

- [ ] **Step 3: Annotate `ContentHandler.List` and `ContentHandler.GetByID`**

```go
// List godoc
// @Summary      List safety content
// @Tags         content
// @Produce      json
// @Param        category     query     string  false  "Filter by category"
// @Param        subcategory  query     string  false  "Filter by subcategory"
// @Success      200          {array}   dto.ContentResponse
// @Failure      500          {object}  dto.ErrorResponse
// @Router       /content [get]
```

```go
// GetByID godoc
// @Summary      Get a safety content item by ID
// @Tags         content
// @Produce      json
// @Param        id   path      string  true  "Content ID"
// @Success      200  {object}  dto.ContentResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /content/{id} [get]
```

- [ ] **Step 4: Annotate `AIHandler.Ask`**

This route is rate limited, hence the 429.

```go
// Ask godoc
// @Summary      Ask the safety assistant a question
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        request  body      dto.AskAIRequest  true  "Question and optional topic"
// @Success      200      {object}  dto.AskAIResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      429      {object}  dto.ErrorResponse  "Rate limited"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /ai/ask [post]
```

- [ ] **Step 5: Annotate `SubmissionHandler.Create`**

Also rate limited.

```go
// Create godoc
// @Summary      Submit a correction to station data
// @Description  Community-reported correction. Enters a pending queue for admin review.
// @Tags         submissions
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateSubmissionRequest  true  "Correction details"
// @Success      201      {object}  dto.MessageResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      429      {object}  dto.ErrorResponse  "Rate limited"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /submissions [post]
```

- [ ] **Step 6: Annotate `AuthHandler.Login` and `AuthHandler.Setup`**

```go
// Login godoc
// @Summary      Authenticate an admin
// @Description  Returns a JWT valid for 24 hours.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest  true  "Credentials"
// @Success      200      {object}  dto.LoginResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse  "Invalid credentials"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /auth/login [post]
```

```go
// Setup godoc
// @Summary      Create the first admin account
// @Description  One-time bootstrap. Returns 403 once any admin exists.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RegisterRequest  true  "Credentials for the first admin"
// @Success      201      {object}  dto.CreatedAdminResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      403      {object}  dto.ErrorResponse  "Setup already completed"
// @Failure      409      {object}  dto.ErrorResponse  "Email already registered"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /auth/setup [post]
```

- [ ] **Step 7: Regenerate and verify all eight paths landed**

```bash
make swagger
python3 -c "
import json
spec = json.load(open('docs/swagger.json'))
want = [('/stations','get'),('/stations/{id}','get'),('/content','get'),('/content/{id}','get'),('/ai/ask','post'),('/submissions','post'),('/auth/login','post'),('/auth/setup','post')]
missing = [(p,m) for p,m in want if m not in spec.get('paths',{}).get(p,{})]
print('missing:', missing if missing else 'none')
print('total paths:', len(spec.get('paths',{})))
"
```

Expected: `missing: none` and `total paths: 8`.

- [ ] **Step 8: Run the full suite**

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: all pass.

- [ ] **Step 9: Commit**

```bash
git add internal/adapter/handler/ docs/
git commit -m "docs(api): annotate the eight public routes"
```

---

## Task 4: Annotate the seven admin routes

**Files:**
- Modify: `internal/adapter/handler/station.go`, `internal/adapter/handler/submission.go`, `internal/adapter/handler/auth.go`
- Regenerate: `docs/`

**Interfaces:**
- Consumes: everything from Tasks 1-3
- Produces: fifteen documented paths in `docs/swagger.json`

Every route here sits behind `middleware.Auth`, so each carries `@Security BearerAuth` and declares 401.

- [ ] **Step 1: Annotate `SubmissionHandler.ListPending` and `Review`**

```go
// ListPending godoc
// @Summary      List pending submissions
// @Tags         admin, submissions
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   dto.SubmissionResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /admin/submissions [get]
```

```go
// Review godoc
// @Summary      Approve or reject a submission
// @Tags         admin, submissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                       true  "Submission UUID"
// @Param        request  body      dto.ReviewSubmissionRequest  true  "Review decision"
// @Success      200      {object}  dto.MessageResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      404      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /admin/submissions/{id} [patch]
```

- [ ] **Step 2: Annotate `AuthHandler.Register` and `List`**

```go
// Register godoc
// @Summary      Create an admin account
// @Tags         admin, auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.RegisterRequest  true  "Credentials"
// @Success      201      {object}  dto.CreatedAdminResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      409      {object}  dto.ErrorResponse  "Email already registered"
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /admin/users [post]
```

```go
// List godoc
// @Summary      List admin accounts
// @Tags         admin, auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   dto.AdminUserResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /admin/users [get]
```

- [ ] **Step 3: Annotate `StationHandler.Create`, `Update`, and `Deactivate`**

```go
// Create godoc
// @Summary      Create a fire station
// @Tags         admin, stations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateStationRequest  true  "Station details, at least one contact required"
// @Success      201      {object}  dto.StationResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /admin/stations [post]
```

```go
// Update godoc
// @Summary      Update a fire station
// @Tags         admin, stations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                    true  "Station UUID"
// @Param        request  body      dto.UpdateStationRequest  true  "Fields to update"
// @Success      200      {object}  dto.MessageResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      404      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /admin/stations/{id} [patch]
```

```go
// Deactivate godoc
// @Summary      Deactivate a fire station
// @Description  Soft delete. The station stops appearing in nearest-station results.
// @Tags         admin, stations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Station UUID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /admin/stations/{id} [delete]
```

- [ ] **Step 4: Regenerate and verify all fifteen routes**

```bash
make swagger
python3 -c "
import json
spec = json.load(open('docs/swagger.json'))
paths = spec.get('paths', {})
count = sum(len([m for m in v if m in ('get','post','patch','put','delete')]) for v in paths.values())
print('path items:', len(paths), '| operations:', count)
secured = [(p,m) for p,v in paths.items() for m,op in v.items() if 'security' in op]
print('secured operations:', len(secured))
print('has BearerAuth definition:', 'BearerAuth' in spec.get('securityDefinitions', {}))
"
```

Expected: `operations: 15`, `secured operations: 7`, `has BearerAuth definition: True`.

- [ ] **Step 5: Run the full suite**

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/handler/ docs/
git commit -m "docs(api): annotate the seven admin routes"
```

---

## Task 5: Route-drift test

**Files:**
- Modify: `tests/swagger_test.go`

**Interfaces:**
- Consumes: the fully annotated `docs/swagger.json` from Task 4; `newTestAppWithEnv` from Task 2
- Produces: a test that fails whenever a registered route is missing from the committed spec

This is the point of the exercise. Generated documentation rots silently — someone adds an endpoint, forgets the annotation, and the spec quietly lies. The test needs no CLI at run time.

- [ ] **Step 1: Write the test**

Append to `tests/swagger_test.go`:

```go
import (
	"encoding/json"
	"os"
	"strings"
)

// swaggerPaths reads the committed spec and returns the set of documented
// "METHOD /path" pairs.
func swaggerPaths(t *testing.T) map[string]bool {
	t.Helper()

	blob, err := os.ReadFile("../docs/swagger.json")
	if err != nil {
		t.Fatalf("cannot read committed spec — run `make swagger`: %v", err)
	}

	var spec struct {
		BasePath string                                       `json:"basePath"`
		Paths    map[string]map[string]json.RawMessage        `json:"paths"`
	}
	if err := json.Unmarshal(blob, &spec); err != nil {
		t.Fatalf("spec is not valid JSON: %v", err)
	}

	documented := make(map[string]bool)
	for path, methods := range spec.Paths {
		for method := range methods {
			documented[strings.ToUpper(method)+" "+spec.BasePath+path] = true
		}
	}
	return documented
}

// ginPathToOpenAPI rewrites Gin's ":id" wildcards as OpenAPI's "{id}".
func ginPathToOpenAPI(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			segments[i] = "{" + seg[1:] + "}"
		}
	}
	return strings.Join(segments, "/")
}

func TestSwaggerSpec_DocumentsEveryRegisteredRoute(t *testing.T) {
	documented := swaggerPaths(t)
	app := newTestApp()

	var undocumented []string
	for _, route := range app.router.Routes() {
		// The docs UI itself is deliberately absent from the spec.
		if strings.HasPrefix(route.Path, "/swagger") {
			continue
		}
		key := route.Method + " " + ginPathToOpenAPI(route.Path)
		if !documented[key] {
			undocumented = append(undocumented, key)
		}
	}

	if len(undocumented) > 0 {
		t.Errorf("routes registered but missing from docs/swagger.json — add annotations and run `make swagger`:\n  %s",
			strings.Join(undocumented, "\n  "))
	}
}

func TestSwaggerSpec_DocumentsNoPhantomRoutes(t *testing.T) {
	documented := swaggerPaths(t)
	app := newTestApp()

	registered := make(map[string]bool)
	for _, route := range app.router.Routes() {
		registered[route.Method+" "+ginPathToOpenAPI(route.Path)] = true
	}

	var phantom []string
	for key := range documented {
		if !registered[key] {
			phantom = append(phantom, key)
		}
	}

	if len(phantom) > 0 {
		t.Errorf("docs/swagger.json documents routes that are not registered:\n  %s",
			strings.Join(phantom, "\n  "))
	}
}
```

Merge the new imports into the existing `import` block rather than adding a second one.

- [ ] **Step 2: Run the test to verify it passes**

```bash
go test ./tests/ -run TestSwaggerSpec -v
```

Expected: PASS for both. If `DocumentsEveryRegisteredRoute` fails, an annotation is missing or `make swagger` was not re-run after Task 4. If `DocumentsNoPhantomRoutes` fails, an `@Router` path does not match its real route — most likely a `/v1` prefix wrongly included in the annotation.

- [ ] **Step 3: Prove the test actually discriminates**

Temporarily comment out the `@Router /content [get]` line in `ContentHandler.List`, regenerate, and confirm the test fails:

```bash
make swagger && go test ./tests/ -run TestSwaggerSpec_DocumentsEveryRegisteredRoute
```

Expected: FAIL naming `GET /v1/content`. Then restore the line, regenerate, and confirm it passes again:

```bash
make swagger && go test ./tests/ -run TestSwaggerSpec
```

Expected: PASS. Confirm `git diff --stat docs/` is empty afterwards — the regenerated spec must be byte-identical to the committed one.

- [ ] **Step 4: Run the full suite**

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: all pass.

- [ ] **Step 5: Verify the UI serves against the running stack**

```bash
docker compose up -d --build api && sleep 6
curl -s -o /dev/null -w "swagger UI: %{http_code}\n" http://localhost:9000/swagger/index.html
curl -s -o /dev/null -w "spec JSON:  %{http_code}\n" http://localhost:9000/swagger/doc.json
```

Expected: both 200. The compose environment is `development`, so the route is mounted.

- [ ] **Step 6: Commit**

```bash
git add tests/swagger_test.go
git commit -m "test(docs): fail when a route is undocumented or documented but unregistered"
```

---

## Self-review notes

- **Spec coverage.** Decision 1 (branch) → Global Constraints. Decision 2 (gating) → Task 2 Steps 5-8. Decision 3 (all 15 routes) → Tasks 3-4. Decision 4 (type success responses, leave errors) → Task 1 Step 6, with the constraint restated globally. Decision 5 (commit generated output) → Task 2 Step 10 and Tasks 3-4 commits. Response types → Task 1. Annotations → Tasks 3-4. Generation and serving → Task 2. Both spec'd tests → Task 2 Step 7 and Task 5.
- **Deviation from the spec.** The spec listed 7 handler edits and named only `loginResponse` for promotion. This plan makes **9**, additionally moving `loginRequest` → `dto.LoginRequest` and `registerRequest` → `dto.RegisterRequest`. Reason: both are unexported in the handler package, so swaggo would emit schemas named `handler.loginRequest`, and every other request type in this API already lives in `dto`. Same principle the spec applied to `loginResponse`, applied consistently.
- **Addition beyond the spec.** Task 5 adds a second drift test, `DocumentsNoPhantomRoutes`, catching the inverse error — an `@Router` path that matches no real route, the most likely form being a wrongly-included `/v1` prefix. Cheap, and it guards the one annotation mistake this plan's own instructions could induce.
- **Type consistency.** `dto.ErrorResponse`, `dto.MessageResponse`, `dto.CreatedAdminResponse`, `dto.LoginResponse`, `dto.LoginRequest`, `dto.RegisterRequest` are defined in Task 1 and referenced by those exact names throughout Tasks 3-4. Pre-existing types referenced in annotations — `dto.StationResponse`, `dto.ContentResponse`, `dto.AskAIRequest`, `dto.AskAIResponse`, `dto.CreateSubmissionRequest`, `dto.ReviewSubmissionRequest`, `dto.SubmissionResponse`, `dto.AdminUserResponse`, `dto.CreateStationRequest`, `dto.UpdateStationRequest` — were each confirmed present in `internal/adapter/dto/` before writing this plan.
- **Tooling deviation from the spec.** The spec said the CLI would be invoked via `go run github.com/swaggo/swag/cmd/swag@latest`. This plan uses Go 1.24+'s `tool` directive (`go get -tool`, then `go tool swag`) instead, so the generator's version is pinned in `go.mod`. `@latest` would let upstream silently change the committed spec's formatting between regenerations, which would make the Task 5 byte-identical check flap.
- **Not addressed.** The 46 `gin.H` error sites and the 6 in middleware stay untyped by decision. No CI wiring. The spec is not published anywhere. `swag fmt` is not run, so annotation alignment is by hand.
