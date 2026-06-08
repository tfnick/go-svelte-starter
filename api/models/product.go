package models

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tfnick/sqlx"
	"github.com/tfnick/go-svelte-starter/api/db"
	"github.com/tfnick/go-svelte-starter/api/framework/data/namelookup"
	"github.com/tfnick/go-svelte-starter/api/framework/timefmt"
)

// Product 产品模型
type Product struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description,omitempty" db:"description"`
	Price       int64  `json:"price" db:"price"`
	Stock       int    `json:"stock" db:"stock"`
	CreatedAt   string `json:"created_at,omitempty" db:"created_at"`
}

// CreateProduct 创建产品（仅用于开发和测试）
func CreateProduct(ctx context.Context, product *Product) error {
	if product.ID == "" {
		product.ID = uuid.Must(uuid.NewV7()).String()
	}
	product.CreatedAt = timefmt.NowSQLiteDateTime()

	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("数据库不可用: %w", err)
	}

	query := `INSERT INTO products (id, name, description, price, stock, created_at) VALUES (:id, :name, :description, :price, :stock, :created_at)`
	_, err = d.NamedExec(query, product)
	return err
}

// GetProductByID 获取产品
func GetProductByID(ctx context.Context, id string) (*Product, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("数据库不可用: %w", err)
	}

	var product Product
	query := d.Rebind(`SELECT * FROM products WHERE id = ?`)
	err = d.Get(&product, query, id)
	if err != nil {
		return nil, fmt.Errorf("获取产品失败: %w", err)
	}
	return &product, nil
}

func ListProducts(ctx context.Context) ([]Product, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	var products []Product
	if err := d.Select(&products, `SELECT * FROM products ORDER BY id`); err != nil {
		return nil, fmt.Errorf("list products failed: %w", err)
	}
	return products, nil
}

func GetProductDisplayNamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	uniqueIDs := namelookup.UniqueNonEmpty(ids)
	if len(uniqueIDs) == 0 {
		return map[string]string{}, nil
	}

	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	query, args, err := sqlx.In(`SELECT id, name FROM products WHERE id IN (?)`, uniqueIDs)
	if err != nil {
		return nil, fmt.Errorf("build product name query failed: %w", err)
	}
	query = d.Rebind(query)

	var rows []namelookup.Row
	if err := d.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("query product names failed: %w", err)
	}
	return namelookup.RowsToMap(rows), nil
}

// UpdateProductStock 更新产品库存（仅用于开发和测试）
func UpdateProductStock(ctx context.Context, productID string, newStock int) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("数据库不可用: %w", err)
	}

	query := d.Rebind(`UPDATE products SET stock = ? WHERE id = ?`)
	_, err = d.Exec(query, newStock, productID)
	return err
}
