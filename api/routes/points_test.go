package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/routes"
)

func TestPointsSSEStreamsInitialPointsMessage(t *testing.T) {
	setupRouteTestDBs(t)

	router := echo.New()
	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/points/sse?client_id=route-points-sse", nil).WithContext(reqCtx)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)
	fwcontext.SetCurrentUser(c, &models.User{ID: "u1", Name: "Ada"})

	if err := routes.PointsSSE(c); err != nil {
		t.Fatalf("points sse: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", contentType)
	}
	if cacheControl := rec.Header().Get(echo.HeaderCacheControl); cacheControl != "no-cache" {
		t.Fatalf("expected no-cache, got %q", cacheControl)
	}

	body := rec.Body.String()
	expected := []string{
		"data: ",
		`"type":"points"`,
		`"presentation":"refresh"`,
		`"user_id":"u1"`,
		`"client_id":"route-points-sse"`,
		`"balance":0`,
	}
	for _, value := range expected {
		if !strings.Contains(body, value) {
			t.Fatalf("expected %s in SSE body, got %s", value, body)
		}
	}
}

func TestPointsSSERequiresCurrentUser(t *testing.T) {
	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/points/sse", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.PointsSSE(c); err != nil {
		t.Fatalf("points sse: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"message":"not logged in"`) {
		t.Fatalf("expected unauthorized envelope, got %s", rec.Body.String())
	}
}
