package routes_test

import (
	"testing"

	"github.com/tfnick/go-svelte-starter/api/routes"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestCurrentUserResponseIncludesAdminFlag(t *testing.T) {
	response := routes.ToCurrentUserResponse(usecase.UserCo{
		ID:            "019ea0c1-0001-7000-8000-000000000001",
		Name:          "Admin",
		Email:         "admin@example.com",
		EmailVerified: true,
		IsAdmin:       true,
	})

	if !response.IsAdmin {
		t.Fatalf("expected admin flag in current user response: %#v", response)
	}
}

func TestAuthStatusUserResponseIncludesAdminFlag(t *testing.T) {
	response := routes.ToAuthStatusUserResponse(usecase.UserCo{
		ID:      "019ea0c1-0001-7000-8000-000000000001",
		Name:    "Admin",
		IsAdmin: true,
	})

	if !response.IsAdmin {
		t.Fatalf("expected admin flag in auth status response: %#v", response)
	}
}
