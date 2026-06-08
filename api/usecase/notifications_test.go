package usecase_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestTriggerExportToastPublishesAsyncExportRealtimeMessage(t *testing.T) {
	sub := realtime.SubscribeClient("u1", "client-export-toast")
	defer sub.Close()

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	if err := usecase.TriggerExportToast(ctx, usecase.TriggerExportToastCmd{UserID: "u1"}); err != nil {
		t.Fatalf("trigger export toast: %v", err)
	}

	select {
	case raw := <-sub.Messages:
		var message struct {
			Type         string `json:"type"`
			Presentation string `json:"presentation"`
			Payload      struct {
				TaskID  string `json:"task_id"`
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(raw, &message); err != nil {
			t.Fatalf("decode realtime message: %v", err)
		}
		if message.Type != "async_export_task" {
			t.Fatalf("expected async export task type, got %q", message.Type)
		}
		if message.Presentation != "toast" {
			t.Fatalf("expected toast presentation, got %q", message.Presentation)
		}
		if message.Payload.TaskID == "" {
			t.Fatalf("expected generated task id")
		}
		if message.Payload.Status != "completed" {
			t.Fatalf("expected completed status, got %q", message.Payload.Status)
		}
		if message.Payload.Message != "Export completed" {
			t.Fatalf("unexpected toast message: %q", message.Payload.Message)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected export toast realtime message")
	}
}

func TestTriggerExportToastValidatesUserID(t *testing.T) {
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	err := usecase.TriggerExportToast(ctx, usecase.TriggerExportToastCmd{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation code, got %q", fwusecase.CodeOf(err))
	}
}
