package events

import (
	"context"
	"time"

	fwevents "github.com/tfnick/go-svelte-starter/api/framework/events"
	"github.com/tfnick/go-svelte-starter/api/framework/timefmt"
	"github.com/tfnick/go-svelte-starter/api/models"
)

type DurableStore struct{}

func (DurableStore) InsertEvent(ctx context.Context, event fwevents.Event) (string, error) {
	persisted, err := models.InsertDomainEvent(ctx, models.InsertDomainEventCmd{
		ID:            event.ID,
		Topic:         event.Topic,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		PayloadJSON:   event.PayloadJSON,
		MetadataJSON:  event.MetadataJSON,
		OccurredAt:    event.OccurredAt,
	})
	if err != nil {
		return "", err
	}
	return persisted.ID, nil
}

func (DurableStore) InsertDelivery(ctx context.Context, eventID string, subscriber string, messageID string) error {
	_, err := models.InsertDomainEventDelivery(ctx, models.InsertDomainEventDeliveryCmd{
		EventID:    eventID,
		Subscriber: subscriber,
		MessageID:  messageID,
	})
	return err
}

func (DurableStore) LoadEvent(ctx context.Context, id string) (fwevents.Event, error) {
	event, err := models.GetDomainEvent(ctx, id)
	if err != nil {
		return fwevents.Event{}, err
	}

	occurredAt, err := time.Parse(time.RFC3339Nano, event.OccurredAt)
	if err != nil {
		occurredAt = timefmt.NowUTC()
	}

	return fwevents.Event{
		ID:            event.ID,
		Topic:         event.Topic,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		PayloadJSON:   []byte(event.PayloadJSON),
		MetadataJSON:  []byte(event.MetadataJSON),
		OccurredAt:    occurredAt.UTC(),
	}, nil
}

func (DurableStore) MarkDeliveryRunning(ctx context.Context, eventID string, subscriber string) error {
	return models.MarkDomainEventDeliveryRunning(ctx, eventID, subscriber)
}

func (DurableStore) MarkDeliverySucceeded(ctx context.Context, eventID string, subscriber string) error {
	return models.MarkDomainEventDeliverySucceeded(ctx, eventID, subscriber)
}

func (DurableStore) MarkDeliveryFailed(ctx context.Context, eventID string, subscriber string, message string) error {
	return models.MarkDomainEventDeliveryFailed(ctx, eventID, subscriber, message)
}
