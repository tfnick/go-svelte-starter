package routes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tfnick/sqlx"
	"github.com/tfnick/go-svelte-starter/api/db"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/routes"
)

const routeSeedAdminUserID = "019ea0c1-0001-7000-8000-000000000001"

func TestGetAllUsersReturnsPaginatedEnvelope(t *testing.T) {
	setupRouteTestDBs(t)
	appDB, err := db.DefaultManager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedRouteUsersForManagement(t, appDB, 5)

	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users?page=2&page_size=2", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.GetAllUsers(c); err != nil {
		t.Fatalf("get all users: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var envelope struct {
		Success bool                 `json:"success"`
		Data    routes.UsersResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("expected success envelope, got %s", rec.Body.String())
	}
	if len(envelope.Data.Items) != 2 {
		t.Fatalf("expected two user items, got %#v", envelope.Data.Items)
	}
	if envelope.Data.Items[0].ID != "route-user-03" || envelope.Data.Items[1].ID != "route-user-02" {
		t.Fatalf("expected stable page items, got %#v", envelope.Data.Items)
	}
	if envelope.Data.Pagination.TotalItems != 8 || envelope.Data.Pagination.TotalPages != 4 {
		t.Fatalf("unexpected pagination metadata: %#v", envelope.Data.Pagination)
	}
}

func TestGetAllUsersRejectsInvalidPageQuery(t *testing.T) {
	setupRouteTestDBs(t)

	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users?page=0&page_size=10", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.GetAllUsers(c); err != nil {
		t.Fatalf("get all users: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"success":false`) || !strings.Contains(body, `"code":"validation"`) {
		t.Fatalf("expected validation envelope, got %s", body)
	}
}

func TestSetUserActiveReturnsUpdatedUser(t *testing.T) {
	setupRouteTestDBs(t)
	appDB, err := db.DefaultManager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedRouteUsersForManagement(t, appDB, 1)

	router := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/users/route-user-01/active", bytes.NewBufferString(`{"active":false}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("route-user-01")
	fwcontext.SetCurrentUser(c, &models.User{ID: routeSeedAdminUserID, Name: "Operator"})

	if err := routes.SetUserActive(c); err != nil {
		t.Fatalf("set user active: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var envelope struct {
		Success bool                `json:"success"`
		Data    routes.UserResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success || envelope.Data.IsActive {
		t.Fatalf("expected disabled user response, got %s", rec.Body.String())
	}
}

func TestSetUserActiveRejectsCurrentUserDisable(t *testing.T) {
	setupRouteTestDBs(t)

	router := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/users/"+routeSeedAdminUserID+"/active", bytes.NewBufferString(`{"active":false}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(routeSeedAdminUserID)
	fwcontext.SetCurrentUser(c, &models.User{ID: routeSeedAdminUserID, Name: "Operator"})

	if err := routes.SetUserActive(c); err != nil {
		t.Fatalf("set user active: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"code":"validation"`) || !strings.Contains(body, "cannot disable current user") {
		t.Fatalf("expected current user validation envelope, got %s", body)
	}
}

func TestSetUserActiveReturnsNotFoundForMissingUser(t *testing.T) {
	setupRouteTestDBs(t)

	router := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/users/missing-user/active", bytes.NewBufferString(`{"active":false}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("missing-user")
	fwcontext.SetCurrentUser(c, &models.User{ID: routeSeedAdminUserID, Name: "Operator"})

	if err := routes.SetUserActive(c); err != nil {
		t.Fatalf("set user active: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"code":"not_found"`) {
		t.Fatalf("expected not found envelope, got %s", body)
	}
}

func seedRouteUsersForManagement(t *testing.T, appDB *sqlx.DB, count int) {
	t.Helper()

	query := appDB.Rebind(`
		INSERT INTO users (
			id, name, email, password_hash, email_verified, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	for i := 1; i <= count; i++ {
		createdAt := fmt.Sprintf("2030-01-01 00:00:%02d", i)
		_, err := appDB.Exec(query,
			fmt.Sprintf("route-user-%02d", i),
			fmt.Sprintf("Route User %02d", i),
			fmt.Sprintf("route-user%02d@example.com", i),
			"",
			1,
			1,
			createdAt,
			createdAt,
		)
		if err != nil {
			t.Fatalf("insert route user %d: %v", i, err)
		}
	}
}
