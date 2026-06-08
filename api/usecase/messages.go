package usecase

import (
	"strings"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
)

type ListMessagesQry struct {
	Queue string
}

type QueueMessageCo struct {
	ID          string
	Queue       string
	BodyPreview string
	Created     string
	Updated     string
	Timeout     string
	Received    int
	Priority    int
}

func ListMessages(ctx fwusecase.Context, qry ListMessagesQry) ([]QueueMessageCo, error) {
	messages, err := models.ListQueueMessages(ctx.Std(), strings.TrimSpace(qry.Queue))
	if err != nil {
		return nil, fwusecase.E(fwusecase.CodeInternal, "failed to load queue messages", err)
	}

	result := make([]QueueMessageCo, 0, len(messages))
	for i := range messages {
		result = append(result, QueueMessageCo{
			ID:          messages[i].ID,
			Queue:       messages[i].Queue,
			BodyPreview: messages[i].BodyPreview,
			Created:     messages[i].Created,
			Updated:     messages[i].Updated,
			Timeout:     messages[i].Timeout,
			Received:    messages[i].Received,
			Priority:    messages[i].Priority,
		})
	}
	return result, nil
}
