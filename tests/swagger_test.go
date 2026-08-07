package tests

import (
	"net/http"
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
