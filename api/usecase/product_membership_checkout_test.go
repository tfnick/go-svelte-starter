package usecase_test

import (
	"testing"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestProductCRUDStoresCreemAndMembershipSettings(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	product, err := usecase.CreateProduct(ctx, usecase.SaveProductCmd{
		Name:                 "Super Quarter",
		Description:          "Quarterly test membership",
		Price:                2999,
		Currency:             "usd",
		Enabled:              true,
		CreemProductID:       "prod_super_quarter",
		BillingType:          usecase.ProductBillingTypeSubscription,
		MembershipLevel:      usecase.MembershipLevelSuper,
		SubscriptionInterval: usecase.SubscriptionIntervalThreeMonths,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if product.ID == "" || product.Currency != "USD" || !product.Enabled || product.CreemProductID != "prod_super_quarter" {
		t.Fatalf("unexpected created product: %#v", product)
	}

	updated, err := usecase.UpdateProduct(ctx, usecase.SaveProductCmd{
		ID:              product.ID,
		Name:            "Premium Lifetime",
		Enabled:         true,
		CreemProductID:  "prod_premium_lifetime",
		BillingType:     usecase.ProductBillingTypeOneTime,
		MembershipLevel: usecase.MembershipLevelPremium,
	})
	if err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updated.Name != "Premium Lifetime" || updated.SubscriptionInterval != "" || updated.BillingType != usecase.ProductBillingTypeOneTime {
		t.Fatalf("unexpected updated product: %#v", updated)
	}
}

func TestApplyOrderMembershipExtendsSubscriptionFromCurrentExpiry(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	if _, err := appDB.Exec(`INSERT INTO users (id, name, email, password_hash, email_verified, is_active, membership_level, membership_expires_at) VALUES ('member-user', 'Ada', 'ada@example.com', '', 1, 1, 'basic', '2099-01-31 00:00:00')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	seedUsecaseCheckoutProduct(t, appDB, "member-product", "prod_member")
	if _, err := appDB.Exec(`INSERT INTO orders (id, user_id, product_id, amount, status) VALUES ('member-order', 'member-user', 'member-product', 0, 'paid')`); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	membership, applied, err := usecase.ApplyOrderMembership(ctx, usecase.ApplyOrderMembershipCmd{OrderID: "member-order"})
	if err != nil {
		t.Fatalf("apply membership: %v", err)
	}
	if !applied || membership.MembershipLevel != usecase.MembershipLevelPremium || membership.MembershipExpiresAt != "2099-02-28 00:00:00" {
		t.Fatalf("unexpected membership result: applied=%v membership=%#v", applied, membership)
	}

	_, appliedAgain, err := usecase.ApplyOrderMembership(ctx, usecase.ApplyOrderMembershipCmd{OrderID: "member-order"})
	if err != nil {
		t.Fatalf("apply membership again: %v", err)
	}
	if appliedAgain {
		t.Fatalf("expected duplicate membership application to be idempotent")
	}
	var user models.User
	if err := appDB.Get(&user, `SELECT * FROM users WHERE id = 'member-user'`); err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.MembershipLevel != usecase.MembershipLevelPremium || !sameSQLiteInstant(user.MembershipExpiresAt, "2099-02-28 00:00:00") {
		t.Fatalf("unexpected persisted user membership: %#v", user)
	}
}

func TestApplyOrderMembershipMakesOneTimeProductPermanent(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	if _, err := appDB.Exec(`INSERT INTO users (id, name, email, password_hash, email_verified, is_active, membership_level, membership_expires_at) VALUES ('one-time-user', 'Ada', 'ada@example.com', '', 1, 1, 'basic', '2026-01-01 00:00:00')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO products (
			id, name, description, price, currency, stock, enabled, creem_product_id, billing_type,
			membership_level, subscription_interval, created_at, updated_at
		) VALUES (?, 'Lifetime', '', 5000, 'USD', 0, 1, 'prod_lifetime', 'one_time', 'super', '', '2026-01-01 00:00:00', '2026-01-01 00:00:00')
	`), "one-time-product"); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if _, err := appDB.Exec(`INSERT INTO orders (id, user_id, product_id, amount, status) VALUES ('one-time-order', 'one-time-user', 'one-time-product', 0, 'paid')`); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	membership, applied, err := usecase.ApplyOrderMembership(ctx, usecase.ApplyOrderMembershipCmd{OrderID: "one-time-order"})
	if err != nil {
		t.Fatalf("apply membership: %v", err)
	}
	if !applied || membership.MembershipLevel != usecase.MembershipLevelSuper || membership.MembershipExpiresAt != usecase.PermanentMembershipExpiresAt {
		t.Fatalf("unexpected membership result: applied=%v membership=%#v", applied, membership)
	}
}

func TestCancelOrderSubscriptionOnlyUpdatesOrderSubscriptionStatus(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	if _, err := appDB.Exec(`INSERT INTO users (id, name, email, password_hash, email_verified, is_active, membership_level, membership_expires_at) VALUES ('cancel-user', 'Ada', 'ada@example.com', '', 1, 1, 'premium', '2099-02-28 00:00:00')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	seedUsecaseCheckoutProduct(t, appDB, "cancel-product", "prod_cancel")
	if _, err := appDB.Exec(`INSERT INTO orders (id, user_id, product_id, amount, status, provider_subscription_id, subscription_status) VALUES ('cancel-order', 'cancel-user', 'cancel-product', 0, 'paid', 'sub_cancel', 'active')`); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	if err := usecase.CancelOrderSubscription(ctx, usecase.CancelOrderSubscriptionCmd{ProviderSubscriptionID: "sub_cancel"}); err != nil {
		t.Fatalf("cancel subscription: %v", err)
	}

	var order models.Order
	if err := appDB.Get(&order, `SELECT * FROM orders WHERE id = 'cancel-order'`); err != nil {
		t.Fatalf("load order: %v", err)
	}
	if order.Status != "paid" || order.SubscriptionStatus != usecase.OrderSubscriptionStatusCanceled {
		t.Fatalf("unexpected order after cancellation: %#v", order)
	}
	var user models.User
	if err := appDB.Get(&user, `SELECT * FROM users WHERE id = 'cancel-user'`); err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.MembershipLevel != "premium" || !sameSQLiteInstant(user.MembershipExpiresAt, "2099-02-28 00:00:00") {
		t.Fatalf("membership should not change on cancellation: %#v", user)
	}
}

func sameSQLiteInstant(actual string, expected string) bool {
	if actual == expected {
		return true
	}
	return actual == expected[:10]+"T"+expected[11:]+"Z"
}
