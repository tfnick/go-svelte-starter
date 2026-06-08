package routes

import (
	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	fwrequest "github.com/tfnick/go-svelte-starter/api/framework/http/request"
	httpresponse "github.com/tfnick/go-svelte-starter/api/framework/http/response"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

type CreateOrderRequest struct {
	UserID string                   `json:"user_id"`
	Items  []CreateOrderItemRequest `json:"items"`
}

type CreateOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

type OrderResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
}

type OrderItemResponse struct {
	ID          string `json:"id"`
	OrderID     string `json:"order_id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Price       int64  `json:"price"`
}

type CreateOrderResponse struct {
	Message string        `json:"message"`
	Order   OrderResponse `json:"order"`
}

type PayOrderResponse struct {
	Message string        `json:"message"`
	Order   OrderResponse `json:"order"`
}

type OrderDetailResponse struct {
	Order OrderResponse       `json:"order"`
	Items []OrderItemResponse `json:"items"`
}

type PaginationResponse struct {
	Page        int  `json:"page"`
	PageSize    int  `json:"page_size"`
	TotalItems  int  `json:"total_items"`
	TotalPages  int  `json:"total_pages"`
	HasPrevious bool `json:"has_previous"`
	HasNext     bool `json:"has_next"`
}

type UserOrdersResponse struct {
	Items      []OrderResponse    `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

func ToOrderResponse(order usecase.OrderCo) OrderResponse {
	return OrderResponse{
		ID:        order.ID,
		UserID:    order.UserID,
		UserName:  order.UserName,
		Amount:    order.Amount,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
	}
}

func ToOrderResponses(orders []usecase.OrderCo) []OrderResponse {
	responses := make([]OrderResponse, 0, len(orders))
	for i := range orders {
		responses = append(responses, ToOrderResponse(orders[i]))
	}
	return responses
}

func ToOrderItemResponse(item usecase.OrderItemCo) OrderItemResponse {
	return OrderItemResponse{
		ID:          item.ID,
		OrderID:     item.OrderID,
		ProductID:   item.ProductID,
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		Price:       item.Price,
	}
}

func ToOrderItemResponses(items []usecase.OrderItemCo) []OrderItemResponse {
	responses := make([]OrderItemResponse, 0, len(items))
	for i := range items {
		responses = append(responses, ToOrderItemResponse(items[i]))
	}
	return responses
}

func ToOrderDetailResponse(detail usecase.OrderDetailCo) OrderDetailResponse {
	return OrderDetailResponse{
		Order: ToOrderResponse(detail.Order),
		Items: ToOrderItemResponses(detail.Items),
	}
}

func ToPaginationResponse(page fwusecase.PageResult) PaginationResponse {
	return PaginationResponse{
		Page:        page.Page,
		PageSize:    page.PageSize,
		TotalItems:  page.TotalItems,
		TotalPages:  page.TotalPages,
		HasPrevious: page.HasPrevious,
		HasNext:     page.HasNext,
	}
}

func ToUserOrdersResponse(orders usecase.UserOrdersCo) UserOrdersResponse {
	return UserOrdersResponse{
		Items:      ToOrderResponses(orders.Items),
		Pagination: ToPaginationResponse(orders.Pagination),
	}
}

// CreateOrder creates a pending local order ledger for provider checkout.
func CreateOrder(c echo.Context) error {
	var req CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		return httpresponse.BadRequest(c, "invalid request data")
	}

	ctx := fwcontext.InternalUsecaseContext(c)
	order, err := usecase.CreateOrder(ctx, usecase.CreateOrderCmd{
		UserID: req.UserID,
	})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	return httpresponse.Created(c, CreateOrderResponse{
		Message: "order created",
		Order:   ToOrderResponse(order),
	})
}

// PayOrder marks an order as paid and runs payment side effects in the same transaction.
func PayOrder(c echo.Context) error {
	orderID := c.Param("id")

	ctx := fwcontext.InternalUsecaseContext(c)
	order, err := usecase.PayOrder(ctx, usecase.PayOrderCmd{OrderID: orderID})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	return httpresponse.OK(c, PayOrderResponse{
		Message: "order paid",
		Order:   ToOrderResponse(order),
	})
}

// GetUserOrders returns paginated orders for a user.
func GetUserOrders(c echo.Context) error {
	userID := c.Param("user_id")
	page := fwrequest.PageQuery(c)
	ctx := fwcontext.InternalUsecaseContext(c)
	orders, err := usecase.GetUserOrders(ctx, usecase.UserOrdersQry{
		UserID:   userID,
		Page:     page.Page,
		PageSize: page.PageSize,
	})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	return httpresponse.OK(c, ToUserOrdersResponse(orders))
}

// GetOrderDetail returns an order and its items.
func GetOrderDetail(c echo.Context) error {
	orderID := c.Param("id")

	ctx := fwcontext.InternalUsecaseContext(c)
	detail, err := usecase.GetOrderDetail(ctx, usecase.OrderDetailQry{OrderID: orderID})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	return httpresponse.OK(c, ToOrderDetailResponse(detail))
}

// UpdateOrderStatus updates an order status.
func UpdateOrderStatus(c echo.Context) error {
	orderID := c.Param("id")
	var req UpdateOrderStatusRequest
	if err := c.Bind(&req); err != nil {
		return httpresponse.BadRequest(c, "invalid request data")
	}

	ctx := fwcontext.InternalUsecaseContext(c)
	if err := usecase.UpdateOrderStatus(ctx, usecase.UpdateOrderStatusCmd{
		OrderID: orderID,
		Status:  req.Status,
	}); err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	return httpresponse.OKMessage(c, "order status updated")
}
