package usecase_test

import (
	"path/filepath"
	"testing"

	"github.com/tfnick/go-svelte-starter/api/db"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func setupUsecaseOrderTxDB(t *testing.T) *db.DBManager {
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

	return manager
}

func TestCreateOrderRollsBackAppTxIncludingProductStock(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)

	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}

	if _, err := appDB.Exec(`INSERT INTO users (id, name, email, password_hash, email_verified, is_active) VALUES ('u1', 'Ada', 'ada@example.com', '', 1, 1)`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := appDB.Exec(`INSERT INTO products (id, name, description, price, stock) VALUES ('p1', 'Keyboard', '', 1000, 5)`); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if _, err := appDB.Exec(`DROP TABLE order_items`); err != nil {
		t.Fatalf("drop order_items to force insert failure: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	_, err = usecase.CreateOrder(ctx, usecase.CreateOrderCmd{
		UserID: "u1",
		Items: []usecase.CreateOrderItemCmd{
			{ProductID: "p1", Quantity: 2},
		},
	})
	if err == nil {
		t.Fatalf("expected order create failure")
	}
	if fwusecase.CodeOf(err) != fwusecase.CodeInternal {
		t.Fatalf("expected internal error code, got %q", fwusecase.CodeOf(err))
	}

	var orderCount int
	if err := appDB.Get(&orderCount, `SELECT COUNT(*) FROM orders`); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 0 {
		t.Fatalf("expected app transaction rollback, found %d orders", orderCount)
	}

	var stock int
	if err := appDB.Get(&stock, `SELECT stock FROM products WHERE id = 'p1'`); err != nil {
		t.Fatalf("get stock: %v", err)
	}
	if stock != 5 {
		t.Fatalf("expected app transaction rollback to restore stock 5, got %d", stock)
	}
}

func TestCreateOrderRollsBackPartialStockReservationOnStockFailure(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)

	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}

	if _, err := appDB.Exec(`INSERT INTO users (id, name, email, password_hash, email_verified, is_active) VALUES ('u1', 'Ada', 'ada@example.com', '', 1, 1)`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := appDB.Exec(`INSERT INTO products (id, name, description, price, stock) VALUES ('tx-p1', 'Keyboard', '', 1000, 5)`); err != nil {
		t.Fatalf("insert first product: %v", err)
	}
	if _, err := appDB.Exec(`INSERT INTO products (id, name, description, price, stock) VALUES ('tx-p2', 'Mouse', '', 500, 0)`); err != nil {
		t.Fatalf("insert second product: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	_, err = usecase.CreateOrder(ctx, usecase.CreateOrderCmd{
		UserID: "u1",
		Items: []usecase.CreateOrderItemCmd{
			{ProductID: "tx-p1", Quantity: 2},
			{ProductID: "tx-p2", Quantity: 1},
		},
	})
	if err == nil {
		t.Fatalf("expected order create failure")
	}
	if fwusecase.CodeOf(err) != fwusecase.CodeConflict {
		t.Fatalf("expected conflict error code, got %q: %v", fwusecase.CodeOf(err), err)
	}

	var firstStock int
	if err := appDB.Get(&firstStock, `SELECT stock FROM products WHERE id = 'tx-p1'`); err != nil {
		t.Fatalf("get first stock: %v", err)
	}
	if firstStock != 5 {
		t.Fatalf("expected app transaction rollback to restore first product stock 5, got %d", firstStock)
	}

	var orderCount int
	if err := appDB.Get(&orderCount, `SELECT COUNT(*) FROM orders`); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 0 {
		t.Fatalf("expected no order after stock failure, found %d", orderCount)
	}
}
