package usecase

import (
	"github.com/google/uuid"
	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
)

type TriggerExportToastCmd struct {
	UserID string
}

func TriggerExportToast(ctx fwusecase.Context, cmd TriggerExportToastCmd) error {
	if cmd.UserID == "" {
		return fwusecase.E(fwusecase.CodeValidation, "user ID is required", nil)
	}

	err := realtime.Publish(cmd.UserID, realtime.NewAsyncExportTaskMessage(realtime.AsyncExportTaskPayload{
		TaskID:  uuid.Must(uuid.NewV7()).String(),
		Status:  "completed",
		Message: "Export completed",
	}, ""))
	if err != nil {
		return fwusecase.E(fwusecase.CodeInternal, "failed to publish export notification", err)
	}
	return nil
}
