package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestEmbeddedFrontendServesIndexForSpaRoutes(t *testing.T) {
	router := echo.New()
	registerFrontendRoutes(router, false, "http://127.0.0.1:5173")

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<div id="app"></div>`) {
		t.Fatal("expected embedded index.html to be served")
	}
}

func TestFrontendRoutesDoNotSwallowApiPaths(t *testing.T) {
	router := echo.New()
	registerFrontendRoutes(router, false, "http://127.0.0.1:5173")

	req := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `<div id="app"></div>`) {
		t.Fatal("api route should not return frontend index.html")
	}
}
