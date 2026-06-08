package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/routes"
)

func TestTriggerExportToastPublishesNotificationAndReturnsEnvelope(t *testing.T) {
	sub := realtime.SubscribeClient("u1", "route-export-toast")
	defer sub.Close()

	router := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/test-export-toast", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)
	fwcontext.SetCurrentUser(c, &models.User{ID: "u1", Name: "Ada"})

	if err := routes.TriggerExportToast(c); err != nil {
		t.Fatalf("trigger export toast: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"success":true`) || !strings.Contains(body, `"message":"export notification sent"`) {
		t.Fatalf("expected success envelope, got %s", body)
	}

	select {
	case message := <-sub.Messages:
		if !strings.Contains(string(message), `"type":"async_export_task"`) {
			t.Fatalf("expected async export task realtime message, got %s", message)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected realtime message")
	}
}

func TestTriggerExportToastRequiresCurrentUser(t *testing.T) {
	router := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/test-export-toast", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.TriggerExportToast(c); err != nil {
		t.Fatalf("trigger export toast: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"message":"not logged in"`) {
		t.Fatalf("expected unauthorized envelope, got %s", body)
	}
}
