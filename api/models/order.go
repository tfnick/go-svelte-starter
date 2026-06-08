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
	ID        string `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	Amount    int64  `json:"amount" db:"amount"`
	Status    string `json:"status" db:"status"`
	CreatedAt string `json:"created_at,omitempty" db:"created_at"`
}

type OrderItem struct {
	ID        string `json:"id" db:"id"`
	OrderID   string `json:"order_id" db:"order_id"`
	ProductID string `json:"product_id" db:"product_id"`
	Quantity  int    `json:"quantity" db:"quantity"`
	Price     int64  `json:"price" db:"price"`
}

func ReserveProductsForOrder(ctx context.Context, items []OrderItem) error {
	appDB, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("数据库不可用: %w", err)
	}

	reserveSQL := appDB.Rebind(`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`)

	for _, item := range items {
		result, err := appDB.Exec(reserveSQL, item.Quantity, item.ProductID, item.Quantity)
		if err != nil {
			return fmt.Errorf("扣减库存失败: %w", err)
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
		return nil, nil, fmt.Errorf("数据库不可用: %w", err)
	}

	insertOrderSQL := `INSERT INTO orders (id, user_id, amount, status, created_at) VALUES (:id, :user_id, :amount, :status, :created_at)`
	if _, err := appDB.NamedExec(insertOrderSQL, order); err != nil {
		return nil, nil, fmt.Errorf("创建订单失败: %w", err)
	}

	insertItemSQL := `INSERT INTO order_items (id, order_id, product_id, quantity, price) VALUES (:id, :order_id, :product_id, :quantity, :price)`
	if _, err := appDB.NamedExec(insertItemSQL, persistedItems); err != nil {
		return nil, nil, fmt.Errorf("创建订单项失败: %w", err)
	}

	return order, persistedItems, nil
}

func GetOrderByID(ctx context.Context, orderID string) (*Order, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("数据库不可用: %w", err)
	}

	var order Order
	query := d.Rebind(`SELECT * FROM orders WHERE id = ?`)
	err = d.Get(&order, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}
	return &order, nil
}

func CountOrdersByUserID(ctx context.Context, userID string) (int, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return 0, fmt.Errorf("database unavailable: %w", err)
	}

	var total int
	query := d.Rebind(`SELECT COUNT(*) FROM orders WHERE user_id = ?`)
	err = d.Get(&total, query, userID)
	if err != nil {
		return 0, fmt.Errorf("count orders failed: %w", err)
	}
	return total, nil
}

func GetOrdersByUserID(ctx context.Context, userID string, limit int, offset int) ([]Order, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("数据库不可用: %w", err)
	}

	var orders []Order
	query := d.Rebind(`SELECT * FROM orders WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`)
	err = d.Select(&orders, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("获取订单列表失败: %w", err)
	}
	return orders, nil
}

func GetOrderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("数据库不可用: %w", err)
	}

	var items []OrderItem
	query := d.Rebind(`SELECT * FROM order_items WHERE order_id = ?`)
	err = d.Select(&items, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("获取订单项失败: %w", err)
	}
	return items, nil
}

func UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("数据库不可用: %w", err)
	}

	query := d.Rebind(`UPDATE orders SET status = ? WHERE id = ?`)
	result, err := d.Exec(query, status, orderID)
	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found: %w", modelerror.ErrNotFound)
	}
	return nil
}
