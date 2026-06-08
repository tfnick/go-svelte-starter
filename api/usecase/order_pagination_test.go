package usecase_test

import (
	"fmt"
	"testing"

	"github.com/tfnick/sqlx"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestGetUserOrdersReturnsRequestedPageAndMetadata(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	const seedUserID = "019ea0c1-0001-7000-8000-000000000001"
	seedUserOrdersForPagination(t, appDB, seedUserID, 5)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	result, err := usecase.GetUserOrders(ctx, usecase.UserOrdersQry{
		UserID:   seedUserID,
		Page:     2,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("get user orders: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected two orders on page 2, got %d", len(result.Items))
	}
	if result.Items[0].ID != "order-03" || result.Items[1].ID != "order-02" {
		t.Fatalf("expected stable created_at desc page order, got %#v", result.Items)
	}

	page := result.Pagination
	if page.Page != 2 || page.PageSize != 2 || page.TotalItems != 5 || page.TotalPages != 3 {
		t.Fatalf("unexpected pagination metadata: %#v", page)
	}
	if !page.HasPrevious || !page.HasNext {
		t.Fatalf("expected page 2 of 3 to have previous and next: %#v", page)
	}
}

func seedUserOrdersForPagination(t *testing.T, appDB *sqlx.DB, userID string, count int) {
	t.Helper()

	query := appDB.Rebind(`INSERT INTO orders (id, user_id, amount, status, created_at) VALUES (?, ?, ?, ?, ?)`)
	for i := 1; i <= count; i++ {
		_, err := appDB.Exec(query,
			fmt.Sprintf("order-%02d", i),
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
