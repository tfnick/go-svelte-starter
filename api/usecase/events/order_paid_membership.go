package events

import (
	"errors"
	"sync"

	fwevents "github.com/tfnick/go-svelte-starter/api/framework/events"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
)

const OrderPaidMembershipSubscriber = "membership.apply_on_order_paid"

var (
	ErrApplyOrderMembershipFuncMissing = errors.New("apply order membership function is required")

	registerMembershipHandlersOnce sync.Once
	registerMembershipHandlersErr  error
)

type ApplyOrderMembershipCmd struct {
	OrderID string
}

type MembershipResult struct {
	UserID              string
	MembershipLevel     string
	MembershipExpiresAt string
}

type ApplyOrderMembershipFunc func(fwusecase.Context, ApplyOrderMembershipCmd) (MembershipResult, bool, error)

type orderPaidMembershipHandler struct {
	apply ApplyOrderMembershipFunc
}

func RegisterMembershipEventHandlers(apply ApplyOrderMembershipFunc) error {
	if apply == nil {
		return ErrApplyOrderMembershipFuncMissing
	}

	registerMembershipHandlersOnce.Do(func() {
		registerMembershipHandlersErr = fwevents.RegisterTransactional[OrderPaidPayload](fwevents.Subscription{
			Topic:      OrderPaidTopic,
			Subscriber: OrderPaidMembershipSubscriber,
		}, orderPaidMembershipHandler{apply: apply}.Handle)
	})
	return registerMembershipHandlersErr
}

func (h orderPaidMembershipHandler) Handle(txCtx fwusecase.Context, _ fwevents.Event, payload OrderPaidPayload) error {
	_, _, err := h.apply(txCtx, ApplyOrderMembershipCmd{
		OrderID: payload.OrderID,
	})
	return err
}
