package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tfnick/sqlx"
	"github.com/tfnick/go-svelte-starter/api/db"
	"github.com/tfnick/go-svelte-starter/api/routes"
)

func TestListNotificationsReturnsPaginatedFilteredEnvelope(t *testing.T) {
	setupRouteTestDBs(t)
	appDB, err := db.DefaultManager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedRouteNotifications(t, appDB)

	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/notifications?type=sms&phone=138&page=1&page_size=2", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.ListNotifications(c); err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool                         `json:"success"`
		Data    routes.NotificationsResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("expected success envelope, got %s", rec.Body.String())
	}
	if len(envelope.Data.Items) != 2 {
		t.Fatalf("expected two notification items, got %#v", envelope.Data.Items)
	}
	if envelope.Data.Items[0].ID != "route-notification-03" || envelope.Data.Items[0].NotificationTypeLabel != "SMS" {
		t.Fatalf("unexpected first notification: %#v", envelope.Data.Items[0])
	}
	if envelope.Data.Pagination.TotalItems != 3 || envelope.Data.Pagination.TotalPages != 2 {
		t.Fatalf("unexpected pagination metadata: %#v", envelope.Data.Pagination)
	}
}

func TestListNotificationsRejectsInvalidType(t *testing.T) {
	setupRouteTestDBs(t)

	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/notifications?type=push", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.ListNotifications(c); err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"success":false`) || !strings.Contains(body, `"notification type is invalid"`) {
		t.Fatalf("expected validation envelope, got %s", body)
	}
}

func seedRouteNotifications(t *testing.T, appDB *sqlx.DB) {
	t.Helper()

	query := appDB.Rebind(`
		INSERT INTO notifications (
			id, notification_type, source_type, source_id, user_id, recipient_email,
			recipient_phone, title, summary, payload_json, status, last_error,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, '', ?, ?)
	`)
	rows := []struct {
		id               string
		notificationType string
		phone            string
		createdAt        string
	}{
		{"route-notification-01", "sms", "13800000001", "2026-01-01 00:00:01"},
		{"route-notification-02", "sms", "13800000002", "2026-01-01 00:00:02"},
		{"route-notification-03", "sms", "13800000003", "2026-01-01 00:00:03"},
		{"route-notification-04", "email", "13900000004", "2026-01-01 00:00:04"},
	}
	for i, row := range rows {
		_, err := appDB.Exec(query,
			row.id,
			row.notificationType,
			"order",
			fmt.Sprintf("route-order-%02d", i+1),
			"route-user-1",
			fmt.Sprintf("route-%02d@example.com", i+1),
			row.phone,
			fmt.Sprintf("Route notification %02d", i+1),
			"Route summary",
			"skipped",
			row.createdAt,
			row.createdAt,
		)
		if err != nil {
			t.Fatalf("insert route notification %s: %v", row.id, err)
		}
	}
}
