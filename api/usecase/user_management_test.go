package usecase_test

import (
	"fmt"
	"testing"

	"github.com/tfnick/sqlx"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestListUsersReturnsRequestedPageAndMetadata(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedUsersForManagement(t, appDB, 5)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	result, err := usecase.ListUsers(ctx, usecase.ListUsersQry{
		Page:     2,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected two users on page 2, got %d", len(result.Items))
	}
	if result.Items[0].ID != "user-03" || result.Items[1].ID != "user-02" {
		t.Fatalf("expected stable created_at desc page order, got %#v", result.Items)
	}

	page := result.Pagination
	if page.Page != 2 || page.PageSize != 2 || page.TotalItems != 8 || page.TotalPages != 4 {
		t.Fatalf("unexpected pagination metadata: %#v", page)
	}
	if !page.HasPrevious || !page.HasNext {
		t.Fatalf("expected page 2 of 4 to have previous and next: %#v", page)
	}
}

func TestSetUserActiveDisablesAndEnablesUser(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedUsersForManagement(t, appDB, 1)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	disabled, err := usecase.SetUserActive(ctx, usecase.SetUserActiveCmd{
		ID:     "user-01",
		Active: false,
	})
	if err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if disabled.IsActive {
		t.Fatalf("expected disabled user, got %#v", disabled)
	}

	enabled, err := usecase.SetUserActive(ctx, usecase.SetUserActiveCmd{
		ID:     "user-01",
		Active: true,
	})
	if err != nil {
		t.Fatalf("enable user: %v", err)
	}
	if !enabled.IsActive {
		t.Fatalf("expected enabled user, got %#v", enabled)
	}
}

func TestSetUserActiveRejectsDisablingCurrentUser(t *testing.T) {
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	ctx.Actor = fwusecase.ActorContext{
		Authenticated: true,
		UserID:        "user-01",
	}

	_, err := usecase.SetUserActive(ctx, usecase.SetUserActiveCmd{
		ID:     "user-01",
		Active: false,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestSetUserActiveReturnsNotFoundForMissingUser(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)

	_, err := usecase.SetUserActive(ctx, usecase.SetUserActiveCmd{
		ID:     "missing-user",
		Active: false,
	})
	if err == nil {
		t.Fatal("expected not found error")
	}
	if fwusecase.CodeOf(err) != fwusecase.CodeNotFound {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func seedUsersForManagement(t *testing.T, appDB *sqlx.DB, count int) {
	t.Helper()

	query := appDB.Rebind(`
		INSERT INTO users (
			id, name, email, password_hash, email_verified, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	for i := 1; i <= count; i++ {
		createdAt := fmt.Sprintf("2030-01-01 00:00:%02d", i)
		_, err := appDB.Exec(query,
			fmt.Sprintf("user-%02d", i),
			fmt.Sprintf("User %02d", i),
			fmt.Sprintf("user%02d@example.com", i),
			"",
			1,
			1,
			createdAt,
			createdAt,
		)
		if err != nil {
			t.Fatalf("insert user %d: %v", i, err)
		}
	}
}
