package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tfnick/go-svelte-starter/api/db"
	"github.com/tfnick/go-svelte-starter/api/routes"
	"github.com/tfnick/sqlx"
)

func TestGetUserOrdersReturnsPaginatedEnvelope(t *testing.T) {
	setupRouteTestDBs(t)
	appDB, err := db.DefaultManager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	const seedUserID = "019ea0c1-0001-7000-8000-000000000001"
	seedRouteUserOrdersForPagination(t, appDB, seedUserID, 5)

	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/orders/user/"+seedUserID+"?page=2&page_size=2", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)
	c.SetParamNames("user_id")
	c.SetParamValues(seedUserID)

	if err := routes.GetUserOrders(c); err != nil {
		t.Fatalf("get user orders: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var envelope struct {
		Success bool                      `json:"success"`
		Data    routes.UserOrdersResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("expected success envelope, got %s", rec.Body.String())
	}
	if len(envelope.Data.Items) != 2 {
		t.Fatalf("expected two order items, got %#v", envelope.Data.Items)
	}
	if envelope.Data.Items[0].ID != "route-order-03" || envelope.Data.Items[1].ID != "route-order-02" {
		t.Fatalf("expected stable page items, got %#v", envelope.Data.Items)
	}
	if envelope.Data.Pagination.TotalItems != 5 || envelope.Data.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected pagination metadata: %#v", envelope.Data.Pagination)
	}
}

func TestGetUserOrdersRejectsInvalidPageQuery(t *testing.T) {
	setupRouteTestDBs(t)

	const seedUserID = "019ea0c1-0001-7000-8000-000000000001"
	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/orders/user/"+seedUserID+"?page=0&page_size=10", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)
	c.SetParamNames("user_id")
	c.SetParamValues(seedUserID)

	if err := routes.GetUserOrders(c); err != nil {
		t.Fatalf("get user orders: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"success":false`) || !strings.Contains(body, `"code":"validation"`) {
		t.Fatalf("expected validation envelope, got %s", body)
	}
}

func TestCreateOrderAcceptsSelectedProductLedgerRequest(t *testing.T) {
	setupRouteTestDBs(t)

	const seedUserID = "019ea0c1-0001-7000-8000-000000000001"
	appDB, err := db.DefaultManager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedRouteCheckoutProduct(t, appDB, "route-product", "prod_route")

	router := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"user_id":"`+seedUserID+`","product_id":"route-product"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.CreateOrder(c); err != nil {
		t.Fatalf("create order: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool                       `json:"success"`
		Data    routes.CreateOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("expected success envelope, got %s", rec.Body.String())
	}
	if envelope.Data.Order.ID == "" {
		t.Fatalf("expected created order id")
	}
	if envelope.Data.Order.Amount != 0 || envelope.Data.Order.Status != "pending" {
		t.Fatalf("unexpected created order: %#v", envelope.Data.Order)
	}
	if envelope.Data.Order.ProductID != "route-product" || envelope.Data.Order.ProductName != "Route Product" {
		t.Fatalf("expected selected product in order response, got %#v", envelope.Data.Order)
	}
}

func seedRouteUserOrdersForPagination(t *testing.T, appDB *sqlx.DB, userID string, count int) {
	t.Helper()

	query := appDB.Rebind(`INSERT INTO orders (id, user_id, amount, status, created_at) VALUES (?, ?, ?, ?, ?)`)
	for i := 1; i <= count; i++ {
		_, err := appDB.Exec(query,
			fmt.Sprintf("route-order-%02d", i),
			userID,
			int64(i*100),
			"pending",
			fmt.Sprintf("2026-01-01 00:00:%02d", i),
		)
		if err != nil {
			t.Fatalf("insert order %d: %v", i, err)
		}
	}
}

func seedRouteCheckoutProduct(t *testing.T, appDB *sqlx.DB, productID string, creemProductID string) {
	t.Helper()

	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO products (
			id, name, description, price, currency, stock, enabled, creem_product_id,
			billing_type, membership_level, subscription_interval, created_at, updated_at
		) VALUES (?, 'Route Product', 'Route checkout product', 1000, 'USD', 0, 1, ?, 'subscription', 'premium', 'month', '2026-01-01 00:00:00', '2026-01-01 00:00:00')
	`), productID, creemProductID); err != nil {
		t.Fatalf("insert route product: %v", err)
	}
}
