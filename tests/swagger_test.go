package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestSwaggerUI_ServedOutsideProduction(t *testing.T) {
	for _, env := range []string{"", "development", "staging", "test", "DEVELOPMENT"} {
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
	for _, env := range []string{"production", "Production", "PRODUCTION", "prod", "qa"} {
		t.Run("env="+env, func(t *testing.T) {
			app := newTestAppWithEnv(env)
			w := app.request("GET", "/swagger/index.html", nil)

			if w.Code != http.StatusNotFound {
				t.Errorf("expected 404 for environment %q, got %d", env, w.Code)
			}
		})
	}
}

// swaggerPaths reads the committed spec and returns the set of documented
// "METHOD /path" pairs.
func swaggerPaths(t *testing.T) map[string]bool {
	t.Helper()

	blob, err := os.ReadFile("../docs/swagger.json")
	if err != nil {
		t.Fatalf("cannot read committed spec — run `make swagger`: %v", err)
	}

	var spec struct {
		BasePath string                                `json:"basePath"`
		Paths    map[string]map[string]json.RawMessage `json:"paths"`
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

// swaggerSecurity reads the committed spec and returns, for every
// "METHOD path" operation, whether it carries a non-empty "security" array.
func swaggerSecurity(t *testing.T) map[string]bool {
	t.Helper()

	blob, err := os.ReadFile("../docs/swagger.json")
	if err != nil {
		t.Fatalf("cannot read committed spec — run `make swagger`: %v", err)
	}

	var spec struct {
		Paths map[string]map[string]struct {
			Security []map[string][]string `json:"security"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(blob, &spec); err != nil {
		t.Fatalf("spec is not valid JSON: %v", err)
	}

	secured := make(map[string]bool)
	for path, methods := range spec.Paths {
		for method, op := range methods {
			secured[strings.ToUpper(method)+" "+path] = len(op.Security) > 0
		}
	}
	return secured
}

func TestSwaggerSpec_AdminRoutesRequireAuth(t *testing.T) {
	secured := swaggerSecurity(t)

	for key, hasSecurity := range secured {
		// key is "METHOD /path" — path already includes the /admin prefix
		// when present, without the basePath.
		parts := strings.SplitN(key, " ", 2)
		path := parts[1]

		if strings.HasPrefix(path, "/admin") {
			if !hasSecurity {
				t.Errorf("admin route %q is missing a @Security annotation in docs/swagger.json", key)
			}
		} else {
			if hasSecurity {
				t.Errorf("non-admin route %q unexpectedly carries a security requirement in docs/swagger.json", key)
			}
		}
	}
}
