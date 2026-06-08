package events

import (
	"context"
	"errors"
	"fmt"
	"sync"

	fwevents "github.com/tfnick/go-svelte-starter/api/framework/events"
	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
)

const (
	OrderPaidTopic      = "order.paid"
	OrderPaidSubscriber = "points.award_on_order_paid"
	OrderPaidPoints     = int64(10)
)

var (
	ErrOrderPaidPointsSubscriberMissing = errors.New("order paid points subscriber is not registered")
	ErrAwardOrderPaidPointsFuncMissing  = errors.New("award order paid points function is required")

	registerEventHandlersOnce sync.Once
	registerEventHandlersErr  error
)

type OrderPaidPayload struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
	Amount  int64  `json:"amount"`
	Points  int64  `json:"points"`
}

type AwardOrderPaidPointsCmd struct {
	UserID  string
	OrderID string
	Points  int64
}

type PointsResult struct {
	UserID  string
	Balance int64
}

type AwardOrderPaidPointsFunc func(fwusecase.Context, AwardOrderPaidPointsCmd) (PointsResult, bool, error)

type orderPaidPointsHandler struct {
	award AwardOrderPaidPointsFunc
}

func RegisterEventHandlers(award AwardOrderPaidPointsFunc) error {
	if award == nil {
		return ErrAwardOrderPaidPointsFuncMissing
	}

	registerEventHandlersOnce.Do(func() {
		registerEventHandlersErr = fwevents.RegisterTransactional[OrderPaidPayload](fwevents.Subscription{
			Topic:      OrderPaidTopic,
			Subscriber: OrderPaidSubscriber,
		}, orderPaidPointsHandler{award: award}.Handle)
	})
	return registerEventHandlersErr
}

func NewOrderPaidEvent(order *models.Order) (fwevents.Event, error) {
	if order == nil {
		return fwevents.Event{}, fmt.Errorf("order is nil")
	}

	return fwevents.NewPayloadEvent(OrderPaidTopic, "order", order.ID, OrderPaidPayload{
		OrderID: order.ID,
		UserID:  order.UserID,
		Amount:  order.Amount,
		Points:  OrderPaidPoints,
	})
}

func (h orderPaidPointsHandler) Handle(txCtx fwusecase.Context, _ fwevents.Event, payload OrderPaidPayload) error {
	points, awarded, err := h.award(txCtx, AwardOrderPaidPointsCmd{
		UserID:  payload.UserID,
		OrderID: payload.OrderID,
		Points:  payload.Points,
	})
	if err != nil {
		return err
	}
	if !awarded {
		return nil
	}

	return fwusecase.RegisterAfterCommit(txCtx, func(context.Context) {
		_ = realtime.Publish(points.UserID, realtime.NewPointsMessage(realtime.PointsPayload{
			UserID:  points.UserID,
			Balance: points.Balance,
		}, ""))
	})
}
