package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/tfnick/go-svelte-starter/api/db"
	"github.com/tfnick/go-svelte-starter/api/framework/data/modelerror"
	"github.com/tfnick/go-svelte-starter/api/framework/timefmt"
)

var ErrInsufficientStock = errors.New("insufficient stock")

type Order struct {
	ID                     string `json:"id" db:"id"`
	UserID                 string `json:"user_id" db:"user_id"`
	ProductID              string `json:"product_id" db:"product_id"`
	Amount                 int64  `json:"amount" db:"amount"`
	Status                 string `json:"status" db:"status"`
	ProviderCheckoutID     string `json:"provider_checkout_id" db:"provider_checkout_id"`
	ProviderOrderID        string `json:"provider_order_id" db:"provider_order_id"`
	ProviderCustomerID     string `json:"provider_customer_id" db:"provider_customer_id"`
	ProviderSubscriptionID string `json:"provider_subscription_id" db:"provider_subscription_id"`
	ProviderProductID      string `json:"provider_product_id" db:"provider_product_id"`
	SubscriptionStatus     string `json:"subscription_status" db:"subscription_status"`
	MembershipAppliedAt    string `json:"membership_applied_at,omitempty" db:"membership_applied_at"`
	CreatedAt              string `json:"created_at,omitempty" db:"created_at"`
}

type OrderProviderRefs struct {
	ProviderCheckoutID     string
	ProviderOrderID        string
	ProviderCustomerID     string
	ProviderSubscriptionID string
	ProviderProductID      string
	SubscriptionStatus     string
}

type OrderItem struct {
	ID        string `json:"id" db:"id"`
	OrderID   string `json:"order_id" db:"order_id"`
	ProductID string `json:"product_id" db:"product_id"`
	Quantity  int    `json:"quantity" db:"quantity"`
	Price     int64  `json:"price" db:"price"`
}

func InsertOrder(ctx context.Context, userID string, amount int64) (*Order, error) {
	return InsertOrderWithProduct(ctx, userID, "", amount)
}

func InsertOrderWithProduct(ctx context.Context, userID string, productID string, amount int64) (*Order, error) {
	order := &Order{
		ID:        uuid.Must(uuid.NewV7()).String(),
		UserID:    userID,
		ProductID: productID,
		Amount:    amount,
		Status:    "pending",
		CreatedAt: timefmt.NowSQLiteDateTime(),
	}

	appDB, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	insertOrderSQL := `
		INSERT INTO orders (id, user_id, product_id, amount, status, created_at)
		VALUES (:id, :user_id, :product_id, :amount, :status, :created_at)
	`
	if _, err := appDB.NamedExec(insertOrderSQL, order); err != nil {
		return nil, fmt.Errorf("create order failed: %w", err)
	}

	return order, nil
}

func ReserveProductsForOrder(ctx context.Context, items []OrderItem) error {
	appDB, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	reserveSQL := appDB.Rebind(`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`)

	for _, item := range items {
		result, err := appDB.Exec(reserveSQL, item.Quantity, item.ProductID, item.Quantity)
		if err != nil {
			return fmt.Errorf("reserve product stock failed: %w", err)
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("%w: product %s", ErrInsufficientStock, item.ProductID)
		}
	}

	return nil
}

func InsertOrderWithItems(ctx context.Context, userID string, items []OrderItem) (*Order, []OrderItem, error) {
	order := &Order{
		ID:        uuid.Must(uuid.NewV7()).String(),
		UserID:    userID,
		Status:    "pending",
		CreatedAt: timefmt.NowSQLiteDateTime(),
	}

	persistedItems := make([]OrderItem, len(items))
	copy(persistedItems, items)

	var totalAmount int64
	for i := range persistedItems {
		persistedItems[i].ID = uuid.Must(uuid.NewV7()).String()
		persistedItems[i].OrderID = order.ID
		totalAmount += persistedItems[i].Price * int64(persistedItems[i].Quantity)
	}
	order.Amount = totalAmount

	appDB, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, nil, fmt.Errorf("database unavailable: %w", err)
	}

	insertOrderSQL := `INSERT INTO orders (id, user_id, amount, status, created_at) VALUES (:id, :user_id, :amount, :status, :created_at)`
	if _, err := appDB.NamedExec(insertOrderSQL, order); err != nil {
		return nil, nil, fmt.Errorf("create order failed: %w", err)
	}

	insertItemSQL := `INSERT INTO order_items (id, order_id, product_id, quantity, price) VALUES (:id, :order_id, :product_id, :quantity, :price)`
	if _, err := appDB.NamedExec(insertItemSQL, persistedItems); err != nil {
		return nil, nil, fmt.Errorf("create order items failed: %w", err)
	}

	return order, persistedItems, nil
}

func GetOrderByID(ctx context.Context, orderID string) (*Order, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	var order Order
	query := d.Rebind(`SELECT * FROM orders WHERE id = ?`)
	err = d.Get(&order, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order failed: %w", err)
	}
	return &order, nil
}

func CountOrdersByUserID(ctx context.Context, userID string) (int, error) {
	total, err := CountOrders(ctx, OrderQuery{UserID: userID})
	if err != nil {
		return 0, err
	}
	return total, nil
}

type OrderQuery struct {
	UserID          string `db:"user_id"`
	Status          string `db:"status"`
	HasCursor       int    `db:"has_cursor"`
	BeforeCreatedAt string `db:"before_created_at"`
	BeforeID        string `db:"before_id"`
	Limit           int    `db:"limit"`
	Offset          int    `db:"offset"`
}

func CountOrders(ctx context.Context, query OrderQuery) (int, error) {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return 0, fmt.Errorf("database unavailable: %w", err)
	}

	var total int
	err = eng.Get(&total, `
		SELECT COUNT(*) FROM orders
		WHERE 1=1
			#[ AND user_id = :user_id ]
			#[ AND status = :status ]
	`, query)
	if err != nil {
		return 0, fmt.Errorf("count orders failed: %w", err)
	}
	return total, nil
}

func GetOrdersByUserID(ctx context.Context, userID string, limit int, offset int) ([]Order, error) {
	return ListOrders(ctx, OrderQuery{UserID: userID, Limit: limit, Offset: offset})
}

func ListOrders(ctx context.Context, query OrderQuery) ([]Order, error) {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	var orders []Order
	err = eng.Select(&orders, `
		SELECT * FROM orders
		WHERE 1=1
			#[ AND user_id = :user_id ]
			#[ AND status = :status ]
			AND (:has_cursor = 0 OR (created_at < :before_created_at OR (created_at = :before_created_at AND id < :before_id)))
		ORDER BY created_at DESC, id DESC
		LIMIT :limit OFFSET :offset
	`, query)
	if err != nil {
		return nil, fmt.Errorf("list orders failed: %w", err)
	}
	return orders, nil
}

func IterateOrders(ctx context.Context, query OrderQuery, batchSize int, handle func([]Order) error) error {
	if handle == nil {
		return fmt.Errorf("order iterator handler is required")
	}
	if batchSize <= 0 {
		batchSize = 1000
	}

	next := query
	next.Limit = batchSize
	next.Offset = 0

	for {
		orders, err := ListOrders(ctx, next)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			return nil
		}

		if err := handle(orders); err != nil {
			return err
		}

		if len(orders) < batchSize {
			return nil
		}
		last := orders[len(orders)-1]
		next.HasCursor = 1
		next.BeforeCreatedAt = last.CreatedAt
		next.BeforeID = last.ID
	}
}

func GetOrderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	var items []OrderItem
	query := d.Rebind(`SELECT * FROM order_items WHERE order_id = ?`)
	err = d.Select(&items, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items failed: %w", err)
	}
	return items, nil
}

func UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`UPDATE orders SET status = ? WHERE id = ?`)
	result, err := d.Exec(query, status, orderID)
	if err != nil {
		return fmt.Errorf("update order status failed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func UpdateOrderProviderRefs(ctx context.Context, orderID string, refs OrderProviderRefs) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`
		UPDATE orders SET
			provider_checkout_id = CASE WHEN ? <> '' THEN ? ELSE provider_checkout_id END,
			provider_order_id = CASE WHEN ? <> '' THEN ? ELSE provider_order_id END,
			provider_customer_id = CASE WHEN ? <> '' THEN ? ELSE provider_customer_id END,
			provider_subscription_id = CASE WHEN ? <> '' THEN ? ELSE provider_subscription_id END,
			provider_product_id = CASE WHEN ? <> '' THEN ? ELSE provider_product_id END,
			subscription_status = CASE WHEN ? <> '' THEN ? ELSE subscription_status END
		WHERE id = ?
	`)
	result, err := d.Exec(query,
		refs.ProviderCheckoutID, refs.ProviderCheckoutID,
		refs.ProviderOrderID, refs.ProviderOrderID,
		refs.ProviderCustomerID, refs.ProviderCustomerID,
		refs.ProviderSubscriptionID, refs.ProviderSubscriptionID,
		refs.ProviderProductID, refs.ProviderProductID,
		refs.SubscriptionStatus, refs.SubscriptionStatus,
		orderID,
	)
	if err != nil {
		return fmt.Errorf("update order provider refs failed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func UpdateOrderSubscriptionStatus(ctx context.Context, orderID string, status string) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`UPDATE orders SET subscription_status = ? WHERE id = ?`)
	result, err := d.Exec(query, status, orderID)
	if err != nil {
		return fmt.Errorf("update order subscription status failed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func UpdateOrderSubscriptionStatusByProviderSubscriptionID(ctx context.Context, providerSubscriptionID string, status string) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`UPDATE orders SET subscription_status = ? WHERE provider_subscription_id = ?`)
	result, err := d.Exec(query, status, providerSubscriptionID)
	if err != nil {
		return fmt.Errorf("update order subscription status failed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func MarkOrderMembershipApplied(ctx context.Context, orderID string, appliedAt string) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`UPDATE orders SET membership_applied_at = ? WHERE id = ?`)
	result, err := d.Exec(query, appliedAt, orderID)
	if err != nil {
		return fmt.Errorf("mark order membership applied failed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func GetOrderByProviderSubscriptionID(ctx context.Context, subscriptionID string) (*Order, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	var order Order
	query := d.Rebind(`SELECT * FROM orders WHERE provider_subscription_id = ?`)
	err = d.Get(&order, query, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("get order by subscription ID failed: %w", err)
	}
	return &order, nil
}

func GetActiveSubscriptionOrdersByUserID(ctx context.Context, userID string, excludeOrderID string) ([]Order, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	var orders []Order
	query := d.Rebind(`
		SELECT o.* FROM orders o
		JOIN products p ON o.product_id = p.id
		WHERE o.user_id = ?
		  AND o.provider_subscription_id IS NOT NULL AND o.provider_subscription_id != ''
		  AND o.subscription_status = 'active'
		  AND o.id != ?
		  AND p.membership_level IN ('premium', 'super')
	`)
	err = d.Select(&orders, query, userID, excludeOrderID)
	if err != nil {
		return nil, fmt.Errorf("get active subscription orders failed: %w", err)
	}
	return orders, nil
}
