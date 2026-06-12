package usecase_test

import (
	"context"
	"testing"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestClearMyTasksClearsOnlyCurrentUsersTerminalTasks(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	seedUsecaseAsyncTask(t, "task-completed", "u1", models.AsyncTaskStatusCompleted)
	seedUsecaseAsyncTask(t, "task-failed", "u1", models.AsyncTaskStatusFailed)
	seedUsecaseAsyncTask(t, "task-processing", "u1", models.AsyncTaskStatusProcessing)
	seedUsecaseAsyncTask(t, "task-other", "u2", models.AsyncTaskStatusCompleted)

	ctx := authenticatedUsecaseContext(t.Context(), "u1", false)
	result, err := usecase.ClearMyTasks(ctx)
	if err != nil {
		t.Fatalf("clear my tasks: %v", err)
	}
	if result.ClearedCount != 2 {
		t.Fatalf("expected two tasks cleared, got %#v", result)
	}

	tasks, err := usecase.ListMyTasks(ctx, usecase.ListMyTasksQry{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list my tasks: %v", err)
	}
	if len(tasks.Items) != 1 || tasks.Items[0].ID != "task-processing" {
		t.Fatalf("expected only processing task visible, got %#v", tasks.Items)
	}

	otherTasks, err := usecase.ListMyTasks(authenticatedUsecaseContext(t.Context(), "u2", false), usecase.ListMyTasksQry{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list other tasks: %v", err)
	}
	if len(otherTasks.Items) != 1 || otherTasks.Items[0].ID != "task-other" {
		t.Fatalf("expected other user's terminal task to remain visible, got %#v", otherTasks.Items)
	}
}

func seedUsecaseAsyncTask(t *testing.T, id string, userID string, status string) {
	t.Helper()

	if err := models.InsertAsyncTask(t.Context(), &models.AsyncTask{
		ID:          id,
		UserID:      userID,
		TaskType:    "test_export",
		Status:      status,
		PayloadJSON: "{}",
		ResultJSON:  "{}",
	}); err != nil {
		t.Fatalf("insert async task %s: %v", id, err)
	}
}

func authenticatedUsecaseContext(ctx context.Context, userID string, admin bool) fwusecase.Context {
	ucCtx := fwusecase.NewContext(ctx, fwusecase.SurfaceInternalAPI)
	ucCtx.Actor = fwusecase.ActorContext{
		Authenticated: true,
		UserID:        userID,
		IsAdmin:       admin,
	}
	return ucCtx
}
