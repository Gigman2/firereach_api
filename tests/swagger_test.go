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
