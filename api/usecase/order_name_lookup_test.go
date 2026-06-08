package usecase_test

import (
	"context"
	"testing"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestOrderUsecaseResolvesDisplayNamesWithFrameworkLookup(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	const seedUserID = "019ea0c1-0001-7000-8000-000000000001"
	const seedProductID = "019ea0c1-0004-7000-8000-000000000001"

	expectedUser, err := models.GetUserByID(context.Background(), seedUserID)
	if err != nil {
		t.Fatalf("load expected user: %v", err)
	}
	expectedProduct, err := models.GetProductByID(context.Background(), seedProductID)
	if err != nil {
		t.Fatalf("load expected product: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	created, err := usecase.CreateOrder(ctx, usecase.CreateOrderCmd{
		UserID: seedUserID,
		Items: []usecase.CreateOrderItemCmd{
			{ProductID: seedProductID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if created.UserName != expectedUser.Name {
		t.Fatalf("expected created order user name %q, got %q", expectedUser.Name, created.UserName)
	}

	detail, err := usecase.GetOrderDetail(ctx, usecase.OrderDetailQry{OrderID: created.ID})
	if err != nil {
		t.Fatalf("get order detail: %v", err)
	}
	if detail.Order.UserName != expectedUser.Name {
		t.Fatalf("expected detail order user name %q, got %q", expectedUser.Name, detail.Order.UserName)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("expected one detail item, got %d", len(detail.Items))
	}
	if detail.Items[0].ProductName != expectedProduct.Name {
		t.Fatalf("expected detail item product name %q, got %q", expectedProduct.Name, detail.Items[0].ProductName)
	}
}
