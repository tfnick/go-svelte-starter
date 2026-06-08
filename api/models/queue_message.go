package models

import (
	"context"
	"fmt"

	"github.com/tfnick/go-svelte-starter/api/db"
)

type QueueMessage struct {
	ID          string `db:"id"`
	Queue       string `db:"queue"`
	BodyPreview string `db:"body_preview"`
	Created     string `db:"created"`
	Updated     string `db:"updated"`
	Timeout     string `db:"timeout"`
	Received    int    `db:"received"`
	Priority    int    `db:"priority"`
}

func ListQueueMessages(ctx context.Context, queueName string) ([]QueueMessage, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	var messages []QueueMessage
	if queueName == "" {
		if err := d.Select(&messages, `
			SELECT id, queue, substr(CAST(body AS TEXT), 1, 240) AS body_preview, created, updated, timeout, received, priority
			FROM goqite
			ORDER BY created DESC
		`); err != nil {
			return nil, fmt.Errorf("list queue messages failed: %w", err)
		}
		return messages, nil
	}

	query := d.Rebind(`
		SELECT id, queue, substr(CAST(body AS TEXT), 1, 240) AS body_preview, created, updated, timeout, received, priority
		FROM goqite
		WHERE queue = ?
		ORDER BY created DESC
	`)
	if err := d.Select(&messages, query, queueName); err != nil {
		return nil, fmt.Errorf("list queue messages failed: %w", err)
	}
	return messages, nil
}
