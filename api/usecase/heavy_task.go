package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/tfnick/go-svelte-starter/api/framework/queue"
	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
)

type EnqueueHeavyTaskCmd struct {
	UserID      string
	TaskType    string
	PayloadJSON string
}

type EnqueueHeavyTaskResult struct {
	TaskID string
}

type HeavyTaskMessage struct {
	TaskID   string `json:"task_id"`
	TaskType string `json:"task_type"`
	UserID   string `json:"user_id"`
}

type ListMyTasksQry struct {
	Page     int
	PageSize int
}

type TaskCo struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	TaskType     string `json:"task_type"`
	Status       string `json:"status"`
	PayloadJSON  string `json:"payload_json"`
	ResultJSON   string `json:"result_json"`
	ErrorMessage string `json:"error_message"`
	RetryCount   int    `json:"retry_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type TasksCo struct {
	Items      []TaskCo          `json:"items"`
	Pagination fwusecase.PageResult `json:"pagination"`
}

func EnqueueHeavyTask(ctx fwusecase.Context, cmd EnqueueHeavyTaskCmd) (EnqueueHeavyTaskResult, error) {
	if cmd.UserID == "" {
		return EnqueueHeavyTaskResult{}, fwusecase.E(fwusecase.CodeValidation, "user ID is required", nil)
	}
	if cmd.TaskType == "" {
		return EnqueueHeavyTaskResult{}, fwusecase.E(fwusecase.CodeValidation, "task type is required", nil)
	}

	payload := cmd.PayloadJSON
	if payload == "" {
		payload = "{}"
	}
	if !json.Valid([]byte(payload)) {
		return EnqueueHeavyTaskResult{}, fwusecase.E(fwusecase.CodeValidation, "payload must be valid JSON", nil)
	}

	taskID := uuid.Must(uuid.NewV7()).String()

	task := &models.AsyncTask{
		ID:          taskID,
		UserID:      cmd.UserID,
		TaskType:    cmd.TaskType,
		Status:      models.AsyncTaskStatusQueued,
		PayloadJSON: payload,
	}

	err := fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
		if err := models.InsertAsyncTask(txCtx.Std(), task); err != nil {
			return err
		}

		msgID, err := DefaultQueueManager.SendJSON(txCtx.Std(), queue.SendOptions{
			Queue: queue.QueueHeavyTasks,
		}, HeavyTaskMessage{
			TaskID:   taskID,
			TaskType: cmd.TaskType,
			UserID:   cmd.UserID,
		})
		if err != nil {
			return fmt.Errorf("enqueue heavy task: %w", err)
		}

		_ = msgID
		return nil
	})
	if err != nil {
		return EnqueueHeavyTaskResult{}, err
	}

	return EnqueueHeavyTaskResult{TaskID: taskID}, nil
}

func HandleHeavyTaskMessage(ctx context.Context, message []byte) error {
	var msg HeavyTaskMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("unmarshal heavy task message: %w", err)
	}

	task, err := models.GetAsyncTaskByID(ctx, msg.TaskID)
	if err != nil {
		return fmt.Errorf("get async task: %w", err)
	}

	if task.Status == models.AsyncTaskStatusCompleted {
		return nil
	}

	if task.RetryCount >= models.MaxAsyncTaskRetries {
		_ = models.UpdateAsyncTaskStatus(ctx, task.ID, models.AsyncTaskStatusFailed, "{}", "max retries exceeded")
		return nil
	}

	_ = models.UpdateAsyncTaskStatus(ctx, task.ID, models.AsyncTaskStatusProcessing, "{}", "")

	err = executeHeavyTask(ctx, msg)

	if err != nil {
		_ = models.IncrementAsyncTaskRetryCount(ctx, task.ID, err.Error())

		if task.RetryCount+1 >= models.MaxAsyncTaskRetries {
			_ = models.UpdateAsyncTaskStatus(ctx, task.ID, models.AsyncTaskStatusFailed, "{}", err.Error())
			publishTaskSSE(task.UserID, task.ID, models.AsyncTaskStatusFailed, err.Error())
			return nil
		}
		return err
	}

	_ = models.UpdateAsyncTaskStatus(ctx, task.ID, models.AsyncTaskStatusCompleted, "{}", "")
	publishTaskSSE(task.UserID, task.ID, models.AsyncTaskStatusCompleted, "Task completed")
	return nil
}

func executeHeavyTask(ctx context.Context, msg HeavyTaskMessage) error {
	switch msg.TaskType {
	case "test_export":
		return nil
	default:
		return fmt.Errorf("unknown task type: %s", msg.TaskType)
	}
}

func publishTaskSSE(userID string, taskID string, status string, message string) {
	_ = realtime.Publish(userID, realtime.NewMessage("heavy_task", realtime.PresentationToast, map[string]interface{}{
		"task_id": taskID,
		"status":  status,
		"message": message,
	}))
}

func ListMyTasks(ctx fwusecase.Context, qry ListMyTasksQry) (TasksCo, error) {
	userID := ctx.Actor.UserID
	if userID == "" {
		return TasksCo{}, fwusecase.E(fwusecase.CodeUnauthorized, "not logged in", nil)
	}

	pageQry, err := fwusecase.NormalizePageQuery(fwusecase.PageQuery{
		Page:     qry.Page,
		PageSize: qry.PageSize,
	})
	if err != nil {
		return TasksCo{}, err
	}

	limit := pageQry.Limit()
	offset := pageQry.Offset()

	total, err := models.CountAsyncTasksByUser(ctx.Std(), userID)
	if err != nil {
		return TasksCo{}, fmt.Errorf("count async tasks: %w", err)
	}

	tasks, err := models.ListAsyncTasksByUser(ctx.Std(), userID, limit, offset)
	if err != nil {
		return TasksCo{}, fmt.Errorf("list async tasks: %w", err)
	}

	items := make([]TaskCo, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, TaskCo{
			ID:           t.ID,
			UserID:       t.UserID,
			TaskType:     t.TaskType,
			Status:       t.Status,
			PayloadJSON:  t.PayloadJSON,
			ResultJSON:   t.ResultJSON,
			ErrorMessage: t.ErrorMessage,
			RetryCount:   t.RetryCount,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}

	return TasksCo{
		Items:      items,
		Pagination: fwusecase.NewPageResult(pageQry, total),
	}, nil
}
