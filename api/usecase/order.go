package usecase

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tfnick/go-svelte-starter/api/framework/data/modelerror"
	"github.com/tfnick/go-svelte-starter/api/framework/data/namelookup"
	"github.com/tfnick/go-svelte-starter/api/framework/events"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	usecaseevents "github.com/tfnick/go-svelte-starter/api/usecase/events"
	"github.com/tfnick/go-svelte-starter/api/usecase/translate"
)

type CreateOrderCmd struct {
	UserID string
	Items  []CreateOrderItemCmd
}

type CreateOrderItemCmd struct {
	ProductID string
	Quantity  int
}

type UserOrdersQry struct {
	UserID   string
	Page     int
	PageSize int
}

type OrderDetailQry struct {
	OrderID string
}

type UpdateOrderStatusCmd struct {
	OrderID string
	Status  string
}

type PayOrderCmd struct {
	OrderID string
}

type OrderCo struct {
	ID        string
	UserID    string
	UserName  string
	Amount    int64
	Status    string
	CreatedAt string
}

type OrderItemCo struct {
	ID          string
	OrderID     string
	ProductID   string
	ProductName string
	Quantity    int
	Price       int64
}

type OrderDetailCo struct {
	Order OrderCo
	Items []OrderItemCo
}

type UserOrdersCo struct {
	Items      []OrderCo
	Pagination fwusecase.PageResult
}

func CreateOrder(ctx fwusecase.Context, cmd CreateOrderCmd) (OrderCo, error) {
	if cmd.UserID == "" {
		return OrderCo{}, fwusecase.E(fwusecase.CodeValidation, "user ID is required", nil)
	}
	if _, err := GetUser(ctx, UserDetailQry{ID: cmd.UserID}); err != nil {
		if fwusecase.CodeOf(err) == fwusecase.CodeNotFound {
			return OrderCo{}, fwusecase.E(fwusecase.CodeValidation, "user not found", err)
		}
		return OrderCo{}, err
	}

	var order *models.Order

	err := fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
		createdOrder, err := models.InsertOrder(txCtx.Std(), cmd.UserID, 0)
		if err != nil {
			return err
		}
		order = createdOrder
		return nil
	})
	if err != nil {
		return OrderCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to create order", err)
	}

	names, err := resolveOrderNames(ctx, []models.Order{*order}, nil)
	if err != nil {
		return OrderCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order display names", err)
	}

	return orderCoFromModel(order, names), nil
}

func GetUserOrders(ctx fwusecase.Context, qry UserOrdersQry) (UserOrdersCo, error) {
	if qry.UserID == "" {
		return UserOrdersCo{}, fwusecase.E(fwusecase.CodeValidation, "user ID is required", nil)
	}

	pageQuery, err := fwusecase.NormalizePageQuery(fwusecase.PageQuery{
		Page:     qry.Page,
		PageSize: qry.PageSize,
	})
	if err != nil {
		return UserOrdersCo{}, err
	}

	total, err := models.CountOrdersByUserID(ctx.Std(), qry.UserID)
	if err != nil {
		return UserOrdersCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to count orders", err)
	}

	orders, err := models.GetOrdersByUserID(ctx.Std(), qry.UserID, pageQuery.Limit(), pageQuery.Offset())
	if err != nil {
		return UserOrdersCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load orders", err)
	}

	names, err := resolveOrderNames(ctx, orders, nil)
	if err != nil {
		return UserOrdersCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order display names", err)
	}

	responses := make([]OrderCo, 0, len(orders))
	for i := range orders {
		responses = append(responses, orderCoFromModel(&orders[i], names))
	}
	return UserOrdersCo{
		Items:      responses,
		Pagination: fwusecase.NewPageResult(pageQuery, total),
	}, nil
}

func GetOrderDetail(ctx fwusecase.Context, qry OrderDetailQry) (OrderDetailCo, error) {
	if qry.OrderID == "" {
		return OrderDetailCo{}, fwusecase.E(fwusecase.CodeValidation, "order ID is required", nil)
	}

	order, err := models.GetOrderByID(ctx.Std(), qry.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OrderDetailCo{}, fwusecase.E(fwusecase.CodeNotFound, "order not found", err)
		}
		return OrderDetailCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order", err)
	}

	items, err := models.GetOrderItems(ctx.Std(), qry.OrderID)
	if err != nil {
		return OrderDetailCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order items", err)
	}

	names, err := resolveOrderNames(ctx, []models.Order{*order}, items)
	if err != nil {
		return OrderDetailCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order display names", err)
	}

	itemResponses := make([]OrderItemCo, 0, len(items))
	for i := range items {
		itemResponses = append(itemResponses, orderItemCoFromModel(&items[i], names))
	}
	return OrderDetailCo{
		Order: orderCoFromModel(order, names),
		Items: itemResponses,
	}, nil
}

func UpdateOrderStatus(ctx fwusecase.Context, cmd UpdateOrderStatusCmd) error {
	if cmd.OrderID == "" {
		return fwusecase.E(fwusecase.CodeValidation, "order ID is required", nil)
	}
	if !isValidOrderStatus(cmd.Status) {
		return fwusecase.E(fwusecase.CodeValidation, "invalid order status", nil)
	}
	if cmd.Status == "paid" {
		return fwusecase.E(fwusecase.CodeValidation, "use pay order endpoint to mark order paid", nil)
	}

	if err := models.UpdateOrderStatus(ctx.Std(), cmd.OrderID, cmd.Status); err != nil {
		if errors.Is(err, modelerror.ErrNotFound) {
			return fwusecase.E(fwusecase.CodeNotFound, "order not found", err)
		}
		return fwusecase.E(fwusecase.CodeInternal, "failed to update order status", err)
	}
	return nil
}

func PayOrder(ctx fwusecase.Context, cmd PayOrderCmd) (OrderCo, error) {
	if cmd.OrderID == "" {
		return OrderCo{}, fwusecase.E(fwusecase.CodeValidation, "order ID is required", nil)
	}
	if len(events.Subscribers(usecaseevents.OrderPaidTopic)) == 0 {
		return OrderCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to pay order", usecaseevents.ErrOrderPaidPointsSubscriberMissing)
	}

	var paidOrder *models.Order
	err := fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
		order, err := models.GetOrderByID(txCtx.Std(), cmd.OrderID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fwusecase.E(fwusecase.CodeNotFound, "order not found", err)
			}
			return fwusecase.E(fwusecase.CodeInternal, "failed to load order", err)
		}

		if order.Status == "paid" {
			paidOrder = order
			return nil
		}
		if order.Status != "pending" {
			return fwusecase.E(fwusecase.CodeConflict, "only pending orders can be paid", nil)
		}

		if err := models.UpdateOrderStatus(txCtx.Std(), order.ID, "paid"); err != nil {
			if errors.Is(err, modelerror.ErrNotFound) {
				return fwusecase.E(fwusecase.CodeNotFound, "order not found", err)
			}
			return fwusecase.E(fwusecase.CodeInternal, "failed to update order status", err)
		}

		order.Status = "paid"
		paidOrder = order

		event, err := usecaseevents.NewOrderPaidEvent(order)
		if err != nil {
			return fwusecase.E(fwusecase.CodeInternal, "failed to build order paid event", err)
		}
		if err := events.Publish(txCtx, event); err != nil {
			return fwusecase.E(fwusecase.CodeInternal, "failed to queue payment side effects", err)
		}
		return nil
	})
	if err != nil {
		return OrderCo{}, err
	}
	if paidOrder == nil {
		return OrderCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to pay order", fmt.Errorf("paid order result is empty"))
	}

	names, err := resolveOrderNames(ctx, []models.Order{*paidOrder}, nil)
	if err != nil {
		return OrderCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order display names", err)
	}
	return orderCoFromModel(paidOrder, names), nil
}

func isValidOrderStatus(status string) bool {
	switch status {
	case "pending", "paid", "shipped", "completed", "cancelled":
		return true
	default:
		return false
	}
}

func resolveOrderNames(ctx fwusecase.Context, orders []models.Order, items []models.OrderItem) (namelookup.Result, error) {
	return translate.Resolve(ctx.Std(), func(batch *namelookup.Batch) {
		namelookup.Collect(batch, translate.UserDisplayName, orders, func(order models.Order) string {
			return order.UserID
		})
		namelookup.Collect(batch, translate.ProductDisplayName, items, func(item models.OrderItem) string {
			return item.ProductID
		})
	})
}

func orderCoFromModel(order *models.Order, names namelookup.Result) OrderCo {
	if order == nil {
		return OrderCo{}
	}

	return OrderCo{
		ID:        order.ID,
		UserID:    order.UserID,
		UserName:  names.Name(translate.UserDisplayName, order.UserID),
		Amount:    order.Amount,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
	}
}

func orderItemCoFromModel(item *models.OrderItem, names namelookup.Result) OrderItemCo {
	if item == nil {
		return OrderItemCo{}
	}

	return OrderItemCo{
		ID:          item.ID,
		OrderID:     item.OrderID,
		ProductID:   item.ProductID,
		ProductName: names.Name(translate.ProductDisplayName, item.ProductID),
		Quantity:    item.Quantity,
		Price:       item.Price,
	}
}
