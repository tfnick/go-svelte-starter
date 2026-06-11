package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tfnick/go-svelte-starter/api/db"
	"github.com/tfnick/sqlx"
)

func TestMarketingRootServesServerRenderedHTML(t *testing.T) {
	setupMarketingTestDB(t)
	router := echo.New()
	registerMarketingRoutes(router)
	registerFrontendRoutes(router, false, "http://127.0.0.1:5173")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.test"
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `<div id="app"></div>`) {
		t.Fatal("marketing root should not serve the embedded Svelte app")
	}
	if !strings.Contains(body, `<link rel="canonical" href="http://example.test/">`) {
		t.Fatalf("expected canonical marketing URL, got %s", body)
	}
	if !strings.Contains(body, `href="/pricing">Choose a plan</a>`) {
		t.Fatalf("expected primary marketing CTA to point to pricing, got %s", body)
	}
	for _, expected := range []string{
		"Motivation",
		"Ability",
		"Prompt",
		"Trust comes from shipped capabilities",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected B=MAP/proof content %q, got %s", expected, body)
		}
	}
	if !strings.Contains(body, `/app/checkout?product_id=marketing-product`) {
		t.Fatalf("expected product checkout CTA, got %s", body)
	}
	if strings.Contains(body, "customer logo") || strings.Contains(body, "trusted by") {
		t.Fatalf("marketing proof should not use invented social proof, got %s", body)
	}
}

func TestMarketingSEOEndpoints(t *testing.T) {
	setupMarketingTestDB(t)
	router := echo.New()
	registerMarketingRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "example.test"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected robots 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Sitemap: http://example.test/sitemap.xml") {
		t.Fatalf("expected sitemap pointer, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	req.Host = "example.test"
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected sitemap 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{
		"<loc>http://example.test/</loc>",
		"<loc>http://example.test/pricing</loc>",
		"<loc>http://example.test/features</loc>",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected sitemap entry %q, got %s", expected, body)
		}
	}
}

func TestEmbeddedFrontendServesIndexForAppRoutes(t *testing.T) {
	router := echo.New()
	registerFrontendRoutes(router, false, "http://127.0.0.1:5173")

	req := httptest.NewRequest(http.MethodGet, "/app/login", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<div id="app"></div>`) {
		t.Fatal("expected embedded index.html to be served")
	}
}

func TestLegacyFrontendRoutesRedirectToApp(t *testing.T) {
	router := echo.New()
	registerFrontendRoutes(router, false, "http://127.0.0.1:5173")

	req := httptest.NewRequest(http.MethodGet, "/login?redirect=%2Fapp%2Fcheckout%3Fproduct_id%3Dp1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/app/login?redirect=%2Fapp%2Fcheckout%3Fproduct_id%3Dp1" {
		t.Fatalf("unexpected redirect location %q", location)
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

func setupMarketingTestDB(t *testing.T) *db.DBManager {
	t.Helper()

	previous := db.DefaultManager
	manager := db.NewDBManager()
	db.DefaultManager = manager
	dir := t.TempDir()
	t.Cleanup(func() {
		_ = manager.Close()
		db.DefaultManager = previous
	})

	if err := manager.Open("app", "sqlite", filepath.Join(dir, "app.db")); err != nil {
		t.Fatalf("open app db: %v", err)
	}
	if err := manager.AutoMigrate("app"); err != nil {
		t.Fatalf("migrate app db: %v", err)
	}
	if err := manager.Open("shared", "sqlite", filepath.Join(dir, "shared.db")); err != nil {
		t.Fatalf("open shared db: %v", err)
	}
	if err := manager.AutoMigrate("shared"); err != nil {
		t.Fatalf("migrate shared db: %v", err)
	}

	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedMarketingProduct(t, appDB)
	return manager
}

func seedMarketingProduct(t *testing.T, appDB *sqlx.DB) {
	t.Helper()

	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO products (
			id, name, description, price, currency, stock, enabled, creem_product_id,
			billing_type, membership_level, subscription_interval, created_at, updated_at
		) VALUES (?, 'Marketing Plan', 'Checkout-ready marketing plan', 2900, 'USD', 0, 1, 'prod_marketing', 'subscription', 'premium', 'month', '2026-01-01 00:00:00', '2026-01-01 00:00:00')
	`), "marketing-product"); err != nil {
		t.Fatalf("insert marketing product: %v", err)
	}
}
