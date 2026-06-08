package routes

import (
	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	httpresponse "github.com/tfnick/go-svelte-starter/api/framework/http/response"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

type QueueMessageResponse struct {
	ID          string `json:"id"`
	Queue       string `json:"queue"`
	BodyPreview string `json:"body_preview"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	Timeout     string `json:"timeout"`
	Received    int    `json:"received"`
	Priority    int    `json:"priority"`
}

func ListMessages(c echo.Context) error {
	ctx := fwcontext.InternalUsecaseContext(c)
	messages, err := usecase.ListMessages(ctx, usecase.ListMessagesQry{
		Queue: c.QueryParam("queue"),
	})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}
	return httpresponse.OK(c, toQueueMessageResponses(messages))
}

func toQueueMessageResponse(message usecase.QueueMessageCo) QueueMessageResponse {
	return QueueMessageResponse{
		ID:          message.ID,
		Queue:       message.Queue,
		BodyPreview: message.BodyPreview,
		Created:     message.Created,
		Updated:     message.Updated,
		Timeout:     message.Timeout,
		Received:    message.Received,
		Priority:    message.Priority,
	}
}

func toQueueMessageResponses(messages []usecase.QueueMessageCo) []QueueMessageResponse {
	responses := make([]QueueMessageResponse, 0, len(messages))
	for i := range messages {
		responses = append(responses, toQueueMessageResponse(messages[i]))
	}
	return responses
}
