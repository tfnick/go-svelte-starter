package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	"github.com/tfnick/go-svelte-starter/api/framework/http/middleware"
	fwrequest "github.com/tfnick/go-svelte-starter/api/framework/http/request"
	httpresponse "github.com/tfnick/go-svelte-starter/api/framework/http/response"
	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

type EnqueueTaskRequest struct {
	TaskType    string `json:"task_type"`
	PayloadJSON string `json:"payload_json,omitempty"`
}

type EnqueueTaskResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

type TaskResponse struct {
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

type TasksResponse struct {
	Items      []TaskResponse    `json:"items"`
	Pagination paginationResult  `json:"pagination"`
}

type paginationResult struct {
	Page        int  `json:"page"`
	PageSize    int  `json:"page_size"`
	TotalItems  int  `json:"total_items"`
	TotalPages  int  `json:"total_pages"`
	HasPrevious bool `json:"has_previous"`
	HasNext     bool `json:"has_next"`
}

func ToTaskResponse(task usecase.TaskCo) TaskResponse {
	return TaskResponse{
		ID:           task.ID,
		UserID:       task.UserID,
		TaskType:     task.TaskType,
		Status:       task.Status,
		PayloadJSON:  task.PayloadJSON,
		ResultJSON:   task.ResultJSON,
		ErrorMessage: task.ErrorMessage,
		RetryCount:   task.RetryCount,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}

func ToTasksResponse(tasks usecase.TasksCo) TasksResponse {
	items := make([]TaskResponse, 0, len(tasks.Items))
	for _, t := range tasks.Items {
		items = append(items, ToTaskResponse(t))
	}
	return TasksResponse{
		Items: items,
		Pagination: paginationResult{
			Page:        tasks.Pagination.Page,
			PageSize:    tasks.Pagination.PageSize,
			TotalItems:  tasks.Pagination.TotalItems,
			TotalPages:  tasks.Pagination.TotalPages,
			HasPrevious: tasks.Pagination.HasPrevious,
			HasNext:     tasks.Pagination.HasNext,
		},
	}
}

func EnqueueTask(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	if currentUser == nil {
		return httpresponse.Unauthorized(c, "not logged in")
	}

	var req EnqueueTaskRequest
	if err := c.Bind(&req); err != nil {
		return httpresponse.BadRequest(c, "invalid request data")
	}

	ctx := fwcontext.InternalUsecaseContext(c)
	result, err := usecase.EnqueueHeavyTask(ctx, usecase.EnqueueHeavyTaskCmd{
		UserID:      currentUser.ID,
		TaskType:    req.TaskType,
		PayloadJSON: req.PayloadJSON,
	})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	return httpresponse.Created(c, EnqueueTaskResponse{
		TaskID:  result.TaskID,
		Message: "task enqueued",
	})
}

func ListMyTasks(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	if currentUser == nil {
		return httpresponse.Unauthorized(c, "not logged in")
	}

	pageQry := fwrequest.PageQuery(c)

	ctx := fwcontext.InternalUsecaseContext(c)
	tasks, err := usecase.ListMyTasks(ctx, usecase.ListMyTasksQry{
		Page:     pageQry.Page,
		PageSize: pageQry.PageSize,
	})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	return httpresponse.OK(c, ToTasksResponse(tasks))
}

func UserEventsSSE(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	if currentUser == nil {
		return httpresponse.Unauthorized(c, "not logged in")
	}

	clientID := c.QueryParam("client_id")
	if clientID == "" {
		clientID = uuid.Must(uuid.NewV7()).String()
	}

	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "text/event-stream")
	res.Header().Set(echo.HeaderCacheControl, "no-cache")
	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := res.Writer.(http.Flusher)
	if !ok {
		return httpresponse.InternalServerError(c, fmt.Errorf("response writer does not support streaming"), "streaming unsupported")
	}

	sub := realtime.SubscribeClient(currentUser.ID, clientID)
	defer sub.Close()

	res.WriteHeader(http.StatusOK)

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case message, ok := <-sub.Messages:
			if !ok {
				return nil
			}
			fmt.Fprintf(res, "data: %s\n\n", message)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(res, ": keepalive\n\n")
			flusher.Flush()
		case <-c.Request().Context().Done():
			return nil
		}
	}
}
