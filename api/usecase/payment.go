package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tfnick/go-svelte-starter/api/framework/data/modelerror"
	"github.com/tfnick/go-svelte-starter/api/framework/integrations/credentials"
	"github.com/tfnick/go-svelte-starter/api/framework/integrations/providererror"
	"github.com/tfnick/go-svelte-starter/api/framework/queue"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/payment"
)

type CreateOrderPaymentCheckoutCmd struct {
	OrderID string
}

type ReceivePaymentWebhookCmd struct {
	ChannelCode string
	Signature   string
	RawPayload  []byte
}

type PaymentWebhookJobPayload struct {
	ReceiptID string `json:"receipt_id"`
}

type PaymentCheckoutCo struct {
	Order             OrderCo
	CheckoutURL       string
	ProviderPaymentID string
	ChannelCode       string
	InvocationID      string
	Status            string
}

type PaymentWebhookReceiptCo struct {
	ID              string
	Status          string
	Duplicate       bool
	ProviderEventID string
	EventType       string
}

type paymentChannelConfig struct {
	BaseURL    string `json:"base_url"`
	ProductID  string `json:"product_id"`
	SuccessURL string `json:"success_url"`
	Units      int    `json:"units"`
}

type paymentCredentialBundle struct {
	APIKey        string `json:"api_key"`
	WebhookSecret string `json:"webhook_secret"`
}

func CreateOrderPaymentCheckout(ctx fwusecase.Context, cmd CreateOrderPaymentCheckoutCmd) (PaymentCheckoutCo, error) {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeValidation, "order ID is required", nil)
	}

	config, err := models.GetEnabledPaymentConfig(ctx.Std(), models.PaymentConfigQuery{
		Scenario:  models.IntegrationScenarioPayment,
		Operation: payment.OperationCreatePayment,
	})
	if err != nil {
		if errors.Is(err, modelerror.ErrNotFound) {
			return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "payment channel is not configured", err)
		}
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load payment configuration", err)
	}

	order, err := models.GetOrderByID(ctx.Std(), orderID)
	if err != nil {
		return PaymentCheckoutCo{}, orderLoadError(err)
	}
	if order.Status == "paid" {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeConflict, "order is already paid", nil)
	}
	if order.Status != "pending" {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeConflict, "only pending orders can be paid", nil)
	}
	if strings.TrimSpace(order.ProductID) == "" {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeValidation, "order product is required", nil)
	}

	product, err := models.GetProductByID(ctx.Std(), order.ProductID)
	if err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order product", err)
	}
	if product.Enabled != 1 {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeValidation, "product is disabled", nil)
	}
	if strings.TrimSpace(product.CreemProductID) == "" {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeValidation, "product Creem ID is required", nil)
	}

	user, err := models.GetUserByID(ctx.Std(), order.UserID)
	if err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load payment customer", err)
	}

	providerCfg, err := paymentProviderConfig(config)
	if err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "payment channel is not configured", err)
	}

	adapter, ok := registeredPaymentAdapter(config.Channel.AdapterKey)
	if !ok {
		cause := fmt.Errorf("payment adapter not registered: %s", config.Channel.AdapterKey)
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "payment adapter is not configured", cause)
	}

	startedAt := time.Now()
	invocation, err := models.CreateIntegrationInvocation(ctx.Std(), models.CreateIntegrationInvocationCmd{
		Scenario:       models.IntegrationScenarioPayment,
		ChannelID:      config.Channel.ID,
		ChannelCode:    config.Channel.ChannelCode,
		ProviderCode:   config.Channel.ProviderCode,
		Operation:      payment.OperationCreatePayment,
		IdempotencyKey: order.ID,
	})
	if err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to record payment invocation", err)
	}

	result, err := adapter.CreatePayment(ctx.Std(), providerCfg, payment.CreatePaymentRequest{
		OrderID:   order.ID,
		UserID:    order.UserID,
		Amount:    order.Amount,
		Currency:  "USD",
		ProductID: product.CreemProductID,
		Customer: payment.Customer{
			Email: user.Email,
		},
		Metadata: map[string]string{
			"order_id":         order.ID,
			"user_id":          order.UserID,
			"product_id":       order.ProductID,
			"creem_product_id": product.CreemProductID,
			"amount":           fmt.Sprintf("%d", order.Amount),
		},
	})
	if err != nil {
		category, retryable, providerRequestID := providerFailureMetadata(err)
		completeErr := models.CompleteIntegrationInvocation(ctx.Std(), models.CompleteIntegrationInvocationCmd{
			ID:                invocation.ID,
			Status:            models.IntegrationInvocationStatusFailed,
			ProviderRequestID: providerRequestID,
			ErrorCategory:     category,
			Retryable:         retryable,
			DurationMS:        time.Since(startedAt).Milliseconds(),
		})
		if completeErr != nil {
			err = fmt.Errorf("%w; complete payment invocation failed: %v", err, completeErr)
		}
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to create payment checkout", err)
	}

	usageJSON, err := json.Marshal(map[string]interface{}{
		"checkout_status": result.Status,
	})
	if err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to record payment invocation", err)
	}
	if err := models.CompleteIntegrationInvocation(ctx.Std(), models.CompleteIntegrationInvocationCmd{
		ID:                invocation.ID,
		Status:            models.IntegrationInvocationStatusSucceeded,
		ProviderRequestID: result.ProviderRequestID,
		UsageJSON:         string(usageJSON),
		DurationMS:        time.Since(startedAt).Milliseconds(),
	}); err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to record payment invocation", err)
	}
	if err := models.UpdateOrderProviderRefs(ctx.Std(), order.ID, models.OrderProviderRefs{
		ProviderCheckoutID: result.ProviderPaymentID,
		ProviderProductID:  product.CreemProductID,
	}); err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to record payment checkout", err)
	}
	order.ProviderCheckoutID = result.ProviderPaymentID
	order.ProviderProductID = product.CreemProductID

	names, err := resolveOrderNames(ctx, []models.Order{*order}, nil)
	if err != nil {
		return PaymentCheckoutCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order display names", err)
	}
	return PaymentCheckoutCo{
		Order:             orderCoFromModel(order, names),
		CheckoutURL:       result.CheckoutURL,
		ProviderPaymentID: result.ProviderPaymentID,
		ChannelCode:       config.Channel.ChannelCode,
		InvocationID:      invocation.ID,
		Status:            result.Status,
	}, nil
}

func ReceivePaymentWebhook(ctx fwusecase.Context, cmd ReceivePaymentWebhookCmd) (PaymentWebhookReceiptCo, error) {
	channelCode := strings.TrimSpace(cmd.ChannelCode)
	if channelCode == "" {
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeValidation, "payment channel code is required", nil)
	}
	if len(cmd.RawPayload) == 0 {
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeValidation, "payment webhook payload is required", nil)
	}
	if DefaultQueueManager == nil {
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeInternal, "queue manager is not configured", nil)
	}

	config, err := models.GetEnabledPaymentConfig(ctx.Std(), models.PaymentConfigQuery{
		Scenario:    models.IntegrationScenarioPayment,
		ChannelCode: channelCode,
	})
	if err != nil {
		if errors.Is(err, modelerror.ErrNotFound) {
			return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeNotFound, "payment channel not found", err)
		}
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load payment configuration", err)
	}
	if config.Channel.WebhookEnabled != 1 {
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeForbidden, "payment webhook is not enabled", nil)
	}

	providerCfg, err := paymentProviderConfig(config)
	if err != nil {
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeInternal, "payment channel is not configured", err)
	}
	adapter, ok := registeredPaymentAdapter(config.Channel.AdapterKey)
	if !ok {
		cause := fmt.Errorf("payment adapter not registered: %s", config.Channel.AdapterKey)
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeInternal, "payment adapter is not configured", cause)
	}

	normalized, err := adapter.NormalizePaymentWebhook(ctx.Std(), providerCfg, payment.WebhookRequest{
		RawPayload:      cmd.RawPayload,
		Signature:       cmd.Signature,
		VerifySignature: true,
	})
	if err != nil {
		_ = persistFailedPaymentWebhookReceipt(ctx, config, cmd.RawPayload, cmd.Signature, "webhook_verification_failed")
		if providerErr, ok := providererror.From(err); ok && providerErr.Category == providererror.CategoryAuth {
			return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeUnauthorized, "payment webhook signature is invalid", err)
		}
		return PaymentWebhookReceiptCo{}, fwusecase.E(fwusecase.CodeValidation, "payment webhook payload is invalid", err)
	}

	receipt, created, err := persistAndEnqueuePaymentWebhookReceipt(ctx, config, normalized, cmd.RawPayload, cmd.Signature)
	if err != nil {
		return PaymentWebhookReceiptCo{}, err
	}
	status := receipt.Status
	if created {
		status = models.IntegrationWebhookReceiptStatusQueued
	}
	return PaymentWebhookReceiptCo{
		ID:              receipt.ID,
		Status:          status,
		Duplicate:       !created,
		ProviderEventID: normalized.ProviderEventID,
		EventType:       normalized.EventType,
	}, nil
}

func HandlePaymentWebhookJob(ctx context.Context, message []byte) error {
	var payload PaymentWebhookJobPayload
	if err := json.Unmarshal(message, &payload); err != nil {
		return err
	}
	payload.ReceiptID = strings.TrimSpace(payload.ReceiptID)
	if payload.ReceiptID == "" {
		return fwusecase.E(fwusecase.CodeValidation, "payment webhook receipt ID is required", nil)
	}

	receipt, err := models.GetIntegrationWebhookReceiptByID(ctx, payload.ReceiptID)
	if err != nil {
		return err
	}
	if receipt.Status == models.IntegrationWebhookReceiptStatusSucceeded || receipt.Status == models.IntegrationWebhookReceiptStatusIgnored {
		return nil
	}
	if err := models.MarkIntegrationWebhookReceiptProcessing(ctx, receipt.ID); err != nil {
		return err
	}

	config, err := models.GetEnabledPaymentConfig(ctx, models.PaymentConfigQuery{
		Scenario:    models.IntegrationScenarioPayment,
		ChannelCode: receipt.ChannelCode,
	})
	if err != nil {
		_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_config_unavailable")
		return err
	}
	providerCfg, err := paymentProviderConfig(config)
	if err != nil {
		_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_config_invalid")
		return err
	}
	adapter, ok := registeredPaymentAdapter(config.Channel.AdapterKey)
	if !ok {
		err := fmt.Errorf("payment adapter not registered: %s", config.Channel.AdapterKey)
		_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_adapter_unavailable")
		return err
	}

	rawPayload, err := credentials.DecryptString(receipt.PayloadCiphertext)
	if err != nil {
		_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_payload_unavailable")
		return err
	}
	normalized, err := adapter.NormalizePaymentWebhook(ctx, providerCfg, payment.WebhookRequest{
		RawPayload: []byte(rawPayload),
	})
	if err != nil {
		_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_webhook_normalize_failed")
		return nil
	}
	if normalized.BusinessEventType == payment.WebhookEventSubscriptionCanceled {
		ucCtx := fwusecase.NewContext(ctx, fwusecase.SurfaceSystem)
		if err := CancelOrderSubscription(ucCtx, CancelOrderSubscriptionCmd{
			OrderID:                normalized.OrderID,
			ProviderSubscriptionID: normalized.ProviderSubscriptionID,
		}); err != nil {
			if fwusecase.CodeOf(err) == fwusecase.CodeInternal {
				_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_subscription_update_temporary_failed")
				return err
			}
			_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_subscription_update_failed")
			return nil
		}
		return models.MarkIntegrationWebhookReceiptSucceeded(ctx, receipt.ID)
	}
	if normalized.BusinessEventType != payment.WebhookEventPaymentSucceeded {
		return models.MarkIntegrationWebhookReceiptIgnored(ctx, receipt.ID, "payment_webhook_ignored")
	}
	if strings.TrimSpace(normalized.OrderID) == "" {
		_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_order_missing")
		return nil
	}

	ucCtx := fwusecase.NewContext(ctx, fwusecase.SurfaceSystem)
	if _, err := PayOrder(ucCtx, PayOrderCmd{
		OrderID:                normalized.OrderID,
		ProviderCheckoutID:     normalized.ProviderPaymentID,
		ProviderOrderID:        normalized.ProviderOrderID,
		ProviderCustomerID:     normalized.ProviderCustomerID,
		ProviderSubscriptionID: normalized.ProviderSubscriptionID,
		ProviderProductID:      normalized.ProviderProductID,
	}); err != nil {
		if fwusecase.CodeOf(err) == fwusecase.CodeInternal {
			_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_fulfillment_temporary_failed")
			return err
		}
		_ = models.MarkIntegrationWebhookReceiptFailed(ctx, receipt.ID, "payment_fulfillment_failed")
		return nil
	}
	return models.MarkIntegrationWebhookReceiptSucceeded(ctx, receipt.ID)
}

func persistAndEnqueuePaymentWebhookReceipt(ctx fwusecase.Context, config models.IntegrationPaymentConfig, normalized payment.NormalizedWebhook, rawPayload []byte, signature string) (models.IntegrationWebhookReceipt, bool, error) {
	payloadCiphertext, err := credentials.EncryptString(string(rawPayload))
	if err != nil {
		return models.IntegrationWebhookReceipt{}, false, fwusecase.E(fwusecase.CodeInternal, "failed to protect payment webhook payload", err)
	}

	safeSnapshotJSON, err := safePaymentSnapshotJSON(normalized)
	if err != nil {
		return models.IntegrationWebhookReceipt{}, false, fwusecase.E(fwusecase.CodeInternal, "failed to record payment webhook", err)
	}

	idempotencyKey := paymentWebhookIdempotencyKey(normalized.ProviderEventID, rawPayload)
	var receipt models.IntegrationWebhookReceipt
	created := false
	err = fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
		var createErr error
		receipt, created, createErr = models.CreateIntegrationWebhookReceipt(txCtx.Std(), models.CreateIntegrationWebhookReceiptCmd{
			Scenario:          models.IntegrationScenarioPayment,
			ChannelID:         config.Channel.ID,
			ChannelCode:       config.Channel.ChannelCode,
			ProviderCode:      config.Channel.ProviderCode,
			ProviderEventID:   normalized.ProviderEventID,
			IdempotencyKey:    idempotencyKey,
			PayloadHash:       sha256Hex(rawPayload),
			PayloadCiphertext: payloadCiphertext,
			SafeSnapshotJSON:  safeSnapshotJSON,
			HeadersHash:       sha256Hex([]byte(signature)),
			Status:            models.IntegrationWebhookReceiptStatusReceived,
		})
		if createErr != nil {
			return createErr
		}
		if !created {
			return nil
		}

		messageID, err := DefaultQueueManager.SendJSON(txCtx.Std(), queue.SendOptions{
			Queue: queue.QueueIntegrationWebhooks,
		}, PaymentWebhookJobPayload{ReceiptID: receipt.ID})
		if err != nil {
			return err
		}
		return models.MarkIntegrationWebhookReceiptQueued(txCtx.Std(), receipt.ID, messageID)
	})
	if err != nil {
		return models.IntegrationWebhookReceipt{}, false, fwusecase.E(fwusecase.CodeInternal, "failed to queue payment webhook", err)
	}
	return receipt, created, nil
}

func persistFailedPaymentWebhookReceipt(ctx fwusecase.Context, config models.IntegrationPaymentConfig, rawPayload []byte, signature string, errorCode string) error {
	payloadCiphertext, err := credentials.EncryptString(string(rawPayload))
	if err != nil {
		return err
	}
	snapshotJSON, err := json.Marshal(map[string]interface{}{
		"error_code": errorCode,
	})
	if err != nil {
		return err
	}
	_, _, err = models.CreateIntegrationWebhookReceipt(ctx.Std(), models.CreateIntegrationWebhookReceiptCmd{
		Scenario:          models.IntegrationScenarioPayment,
		ChannelID:         config.Channel.ID,
		ChannelCode:       config.Channel.ChannelCode,
		ProviderCode:      config.Channel.ProviderCode,
		IdempotencyKey:    "invalid:" + sha256Hex(rawPayload),
		PayloadHash:       sha256Hex(rawPayload),
		PayloadCiphertext: payloadCiphertext,
		SafeSnapshotJSON:  string(snapshotJSON),
		HeadersHash:       sha256Hex([]byte(signature)),
		Status:            models.IntegrationWebhookReceiptStatusFailed,
	})
	return err
}

func paymentProviderConfig(config models.IntegrationPaymentConfig) (payment.ProviderConfig, error) {
	channelConfig, err := parsePaymentConfigJSON(config.Channel.ConfigJSON)
	if err != nil {
		return payment.ProviderConfig{}, fmt.Errorf("parse payment channel config failed: %w", err)
	}
	operationConfig, err := parsePaymentConfigJSON(config.OperationConfig.ConfigJSON)
	if err != nil {
		return payment.ProviderConfig{}, fmt.Errorf("parse payment operation config failed: %w", err)
	}
	channelConfig = mergePaymentConfig(channelConfig, operationConfig)
	if strings.TrimSpace(channelConfig.BaseURL) == "" {
		return payment.ProviderConfig{}, fmt.Errorf("payment base_url is required")
	}

	secrets, err := paymentCredentialSecrets(config.Credential.ValueText)
	if err != nil {
		return payment.ProviderConfig{}, err
	}
	return payment.ProviderConfig{
		ChannelCode:   config.Channel.ChannelCode,
		ProviderCode:  config.Channel.ProviderCode,
		AdapterKey:    config.Channel.AdapterKey,
		BaseURL:       channelConfig.BaseURL,
		APIKey:        secrets.APIKey,
		WebhookSecret: secrets.WebhookSecret,
		ProductID:     channelConfig.ProductID,
		SuccessURL:    channelConfig.SuccessURL,
		Units:         channelConfig.Units,
	}, nil
}

func parsePaymentConfigJSON(raw string) (paymentChannelConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return paymentChannelConfig{}, nil
	}
	var config paymentChannelConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return paymentChannelConfig{}, err
	}
	return config, nil
}

func mergePaymentConfig(base paymentChannelConfig, override paymentChannelConfig) paymentChannelConfig {
	if strings.TrimSpace(override.BaseURL) != "" {
		base.BaseURL = override.BaseURL
	}
	if strings.TrimSpace(override.ProductID) != "" {
		base.ProductID = override.ProductID
	}
	if strings.TrimSpace(override.SuccessURL) != "" {
		base.SuccessURL = override.SuccessURL
	}
	if override.Units > 0 {
		base.Units = override.Units
	}
	return base
}

func paymentCredentialSecrets(value string) (paymentCredentialBundle, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return paymentCredentialBundle{}, fmt.Errorf("payment credential is empty")
	}

	if strings.HasPrefix(value, "{") {
		var bundle paymentCredentialBundle
		if err := json.Unmarshal([]byte(value), &bundle); err != nil {
			return paymentCredentialBundle{}, fmt.Errorf("parse payment credential bundle failed: %w", err)
		}
		bundle.APIKey = strings.TrimSpace(bundle.APIKey)
		bundle.WebhookSecret = strings.TrimSpace(bundle.WebhookSecret)
		if bundle.APIKey == "" {
			return paymentCredentialBundle{}, fmt.Errorf("payment api_key is required")
		}
		return bundle, nil
	}

	return paymentCredentialBundle{APIKey: value}, nil
}

func safePaymentSnapshotJSON(normalized payment.NormalizedWebhook) (string, error) {
	snapshot := map[string]interface{}{}
	for key, value := range normalized.SafeSnapshot {
		snapshot[key] = value
	}
	snapshot["business_event_type"] = normalized.BusinessEventType
	body, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func paymentWebhookIdempotencyKey(providerEventID string, rawPayload []byte) string {
	providerEventID = strings.TrimSpace(providerEventID)
	if providerEventID != "" {
		return providerEventID
	}
	return "payload:" + sha256Hex(rawPayload)
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func orderLoadError(err error) error {
	if errors.Is(err, modelerror.ErrNotFound) {
		return fwusecase.E(fwusecase.CodeNotFound, "order not found", err)
	}
	if strings.Contains(err.Error(), "sql: no rows") {
		return fwusecase.E(fwusecase.CodeNotFound, "order not found", err)
	}
	return fwusecase.E(fwusecase.CodeInternal, "failed to load order", err)
}
