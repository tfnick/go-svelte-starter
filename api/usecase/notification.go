package usecase

import (
	"encoding/json"
	"strings"

	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	"github.com/tfnick/go-svelte-starter/api/framework/timefmt"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
)

const (
	DictionaryTypeNotificationType = "notification_type"
	NotificationTypeRealtime       = "realtime"
)

type CreateNotificationCmd struct {
	NotificationType string
	SourceType       string
	SourceID         string
	UserID           string
	RecipientEmail   string
	RecipientPhone   string
	Title            string
	Summary          string
	PayloadJSON      string
}

type NotificationsQry struct {
	Page     int
	PageSize int
	Type     string
	Email    string
	Phone    string
}

type NotificationCo struct {
	ID                    string
	NotificationType      string
	NotificationTypeLabel string
	SourceType            string
	SourceID              string
	UserID                string
	RecipientEmail        string
	RecipientPhone        string
	Title                 string
	Summary               string
	PayloadJSON           string
	Status                string
	LastError             string
	SentAt                string
	CreatedAt             string
	UpdatedAt             string
}

type NotificationsCo struct {
	Items      []NotificationCo
	Pagination fwusecase.PageResult
}

func CreateNotification(ctx fwusecase.Context, cmd CreateNotificationCmd) (NotificationCo, error) {
	notification, labels, err := notificationInput(ctx, cmd)
	if err != nil {
		return NotificationCo{}, err
	}

	if notification.NotificationType != NotificationTypeRealtime {
		notification.Status = models.NotificationStatusSkipped
	}

	if err := models.InsertNotification(ctx.Std(), &notification); err != nil {
		return NotificationCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to create notification", err)
	}

	if notification.NotificationType == NotificationTypeRealtime {
		published, err := publishRealtimeNotification(notification)
		if err != nil {
			notification.Status = models.NotificationStatusFailed
			notification.LastError = "failed to publish realtime notification"
			if updateErr := models.UpdateNotificationStatus(ctx.Std(), notification.ID, notification.Status, notification.LastError, ""); updateErr != nil {
				return NotificationCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to update notification status", updateErr)
			}
			return NotificationCo{}, fwusecase.E(fwusecase.CodeInternal, notification.LastError, err)
		}

		notification.Status = models.NotificationStatusSent
		notification.LastError = ""
		notification.SentAt = published
		if err := models.UpdateNotificationStatus(ctx.Std(), notification.ID, notification.Status, notification.LastError, notification.SentAt); err != nil {
			return NotificationCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to update notification status", err)
		}
	}

	return notificationCoFromModel(notification, labels), nil
}

func ListNotifications(ctx fwusecase.Context, qry NotificationsQry) (NotificationsCo, error) {
	pageQuery, err := fwusecase.NormalizePageQuery(fwusecase.PageQuery{
		Page:     qry.Page,
		PageSize: qry.PageSize,
	})
	if err != nil {
		return NotificationsCo{}, err
	}

	labels, err := notificationTypeLabels(ctx)
	if err != nil {
		return NotificationsCo{}, err
	}

	notificationType := strings.TrimSpace(strings.ToLower(qry.Type))
	if notificationType != "" {
		if _, ok := labels[notificationType]; !ok {
			return NotificationsCo{}, fwusecase.E(fwusecase.CodeValidation, "notification type is invalid", nil)
		}
	}

	query := models.NotificationQuery{
		NotificationType: notificationType,
		RecipientEmail:   containsLike(strings.TrimSpace(qry.Email)),
		RecipientPhone:   containsLike(strings.TrimSpace(qry.Phone)),
		Limit:            pageQuery.Limit(),
		Offset:           pageQuery.Offset(),
	}

	totalItems, err := models.CountNotifications(ctx.Std(), query)
	if err != nil {
		return NotificationsCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to count notifications", err)
	}

	notifications, err := models.ListNotifications(ctx.Std(), query)
	if err != nil {
		return NotificationsCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load notifications", err)
	}

	return NotificationsCo{
		Items:      notificationCosFromModels(notifications, labels),
		Pagination: fwusecase.NewPageResult(pageQuery, totalItems),
	}, nil
}

func notificationInput(ctx fwusecase.Context, cmd CreateNotificationCmd) (models.Notification, map[string]string, error) {
	labels, err := notificationTypeLabels(ctx)
	if err != nil {
		return models.Notification{}, nil, err
	}

	notificationType := strings.TrimSpace(strings.ToLower(cmd.NotificationType))
	if notificationType == "" {
		return models.Notification{}, nil, fwusecase.E(fwusecase.CodeValidation, "notification type is required", nil)
	}
	if _, ok := labels[notificationType]; !ok {
		return models.Notification{}, nil, fwusecase.E(fwusecase.CodeValidation, "notification type is invalid", nil)
	}

	userID := strings.TrimSpace(cmd.UserID)
	if notificationType == NotificationTypeRealtime && userID == "" {
		return models.Notification{}, nil, fwusecase.E(fwusecase.CodeValidation, "user ID is required for realtime notification", nil)
	}

	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return models.Notification{}, nil, fwusecase.E(fwusecase.CodeValidation, "notification title is required", nil)
	}

	payloadJSON, err := normalizeNotificationPayloadJSON(cmd.PayloadJSON)
	if err != nil {
		return models.Notification{}, nil, err
	}

	return models.Notification{
		NotificationType: notificationType,
		SourceType:       strings.TrimSpace(cmd.SourceType),
		SourceID:         strings.TrimSpace(cmd.SourceID),
		UserID:           userID,
		RecipientEmail:   strings.TrimSpace(cmd.RecipientEmail),
		RecipientPhone:   strings.TrimSpace(cmd.RecipientPhone),
		Title:            title,
		Summary:          strings.TrimSpace(cmd.Summary),
		PayloadJSON:      payloadJSON,
		Status:           models.NotificationStatusPending,
	}, labels, nil
}

func notificationTypeLabels(ctx fwusecase.Context) (map[string]string, error) {
	batch, err := GetDictionaries(ctx, DictionaryBatchQry{
		Types: []string{DictionaryTypeNotificationType},
	})
	if err != nil {
		return nil, err
	}

	labels := map[string]string{}
	for _, option := range batch.Dictionaries[DictionaryTypeNotificationType] {
		labels[option.Value] = option.Label
	}
	return labels, nil
}

func normalizeNotificationPayloadJSON(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "{}", nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return "", fwusecase.E(fwusecase.CodeValidation, "payload JSON is invalid", err)
	}
	if payload == nil {
		return "", fwusecase.E(fwusecase.CodeValidation, "payload JSON must be an object", nil)
	}

	normalized, err := json.Marshal(payload)
	if err != nil {
		return "", fwusecase.E(fwusecase.CodeInternal, "failed to normalize payload JSON", err)
	}
	return string(normalized), nil
}

func publishRealtimeNotification(notification models.Notification) (string, error) {
	sentAt := timefmt.NowSQLiteDateTime()
	err := realtime.Publish(notification.UserID, realtime.NewNotificationMessage(realtime.NotificationPayload{
		ID:         notification.ID,
		Title:      notification.Title,
		Summary:    notification.Summary,
		SourceType: notification.SourceType,
		SourceID:   notification.SourceID,
	}, ""))
	if err != nil {
		return "", err
	}
	return sentAt, nil
}

func notificationCoFromModel(notification models.Notification, labels map[string]string) NotificationCo {
	label := labels[notification.NotificationType]
	if label == "" {
		label = notification.NotificationType
	}

	return NotificationCo{
		ID:                    notification.ID,
		NotificationType:      notification.NotificationType,
		NotificationTypeLabel: label,
		SourceType:            notification.SourceType,
		SourceID:              notification.SourceID,
		UserID:                notification.UserID,
		RecipientEmail:        notification.RecipientEmail,
		RecipientPhone:        notification.RecipientPhone,
		Title:                 notification.Title,
		Summary:               notification.Summary,
		PayloadJSON:           notification.PayloadJSON,
		Status:                notification.Status,
		LastError:             notification.LastError,
		SentAt:                notification.SentAt,
		CreatedAt:             notification.CreatedAt,
		UpdatedAt:             notification.UpdatedAt,
	}
}

func notificationCosFromModels(notifications []models.Notification, labels map[string]string) []NotificationCo {
	result := make([]NotificationCo, 0, len(notifications))
	for i := range notifications {
		result = append(result, notificationCoFromModel(notifications[i], labels))
	}
	return result
}

func containsLike(value string) string {
	if value == "" {
		return ""
	}
	return "%" + value + "%"
}
