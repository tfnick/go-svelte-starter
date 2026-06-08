package creem

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tfnick/go-svelte-starter/api/framework/integrations/providererror"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/payment"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Adapter struct {
	client HTTPDoer
}

func NewAdapter(client HTTPDoer) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{client: client}
}

type createCheckoutRequest struct {
	ProductID  string            `json:"product_id"`
	RequestID  string            `json:"request_id,omitempty"`
	SuccessURL string            `json:"success_url,omitempty"`
	Customer   *checkoutCustomer `json:"customer,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Units      int               `json:"units,omitempty"`
}

type checkoutCustomer struct {
	Email string `json:"email,omitempty"`
}

type createCheckoutResponse struct {
	ID          string `json:"id"`
	CheckoutURL string `json:"checkout_url"`
	ProductID   string `json:"product_id"`
	Status      string `json:"status"`
}

type webhookEvent struct {
	ID        string                 `json:"id"`
	EventType string                 `json:"eventType"`
	Object    webhookCheckoutObject  `json:"object"`
	Raw       map[string]interface{} `json:"-"`
}

type webhookCheckoutObject struct {
	ID        string                 `json:"id"`
	Object    string                 `json:"object"`
	RequestID string                 `json:"request_id"`
	Status    string                 `json:"status"`
	ProductID string                 `json:"product_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (a *Adapter) CreatePayment(ctx context.Context, cfg payment.ProviderConfig, req payment.CreatePaymentRequest) (payment.CreatePaymentResult, error) {
	if err := validateCreateConfig(cfg); err != nil {
		return payment.CreatePaymentResult{}, err
	}

	body, err := json.Marshal(createCheckoutRequest{
		ProductID:  firstNonEmpty(req.ProductID, cfg.ProductID),
		RequestID:  strings.TrimSpace(req.OrderID),
		SuccessURL: firstNonEmpty(req.SuccessURL, cfg.SuccessURL),
		Customer:   customerFromRequest(req.Customer),
		Metadata:   req.Metadata,
		Units:      firstPositive(req.Units, cfg.Units),
	})
	if err != nil {
		return payment.CreatePaymentResult{}, providererror.New(providererror.CategoryValidation, false, "payment request is invalid", err)
	}

	endpoint, err := checkoutURL(cfg.BaseURL)
	if err != nil {
		return payment.CreatePaymentResult{}, providererror.New(providererror.CategoryValidation, false, "payment base URL is invalid", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return payment.CreatePaymentResult{}, providererror.New(providererror.CategoryTemporary, true, "failed to create payment request", err)
	}
	httpReq.Header.Set("x-api-key", cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return payment.CreatePaymentResult{}, providererror.New(providererror.CategoryTemporary, true, "payment provider request failed", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return payment.CreatePaymentResult{}, providererror.New(providererror.CategoryTemporary, true, "failed to read payment response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return payment.CreatePaymentResult{}, providerErrorFromStatus(resp.StatusCode)
	}

	var parsed createCheckoutResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return payment.CreatePaymentResult{}, providererror.New(providererror.CategoryProviderInternal, true, "payment response is invalid", err)
	}
	if strings.TrimSpace(parsed.CheckoutURL) == "" || strings.TrimSpace(parsed.ID) == "" {
		return payment.CreatePaymentResult{}, providererror.New(providererror.CategoryProviderInternal, true, "payment response is incomplete", nil)
	}

	return payment.CreatePaymentResult{
		ProviderPaymentID: parsed.ID,
		CheckoutURL:       parsed.CheckoutURL,
		Status:            parsed.Status,
		ProviderRequestID: parsed.ID,
	}, nil
}

func (a *Adapter) NormalizePaymentWebhook(_ context.Context, cfg payment.ProviderConfig, req payment.WebhookRequest) (payment.NormalizedWebhook, error) {
	if req.VerifySignature {
		if err := verifySignature(cfg.WebhookSecret, req.RawPayload, req.Signature); err != nil {
			return payment.NormalizedWebhook{}, err
		}
	}

	var event webhookEvent
	if err := json.Unmarshal(req.RawPayload, &event); err != nil {
		return payment.NormalizedWebhook{}, providererror.New(providererror.CategoryValidation, false, "payment webhook payload is invalid", err)
	}

	orderID := strings.TrimSpace(event.Object.RequestID)
	if orderID == "" {
		orderID = metadataString(event.Object.Metadata, "order_id")
	}
	if orderID == "" {
		orderID = metadataString(event.Object.Metadata, "orderId")
	}

	paymentStatus := event.Object.Status
	businessEventType := ""
	if event.EventType == "checkout.completed" {
		paymentStatus = "succeeded"
		businessEventType = payment.WebhookEventPaymentSucceeded
	}

	snapshot := map[string]interface{}{
		"event_type":          event.EventType,
		"provider_event_id":   event.ID,
		"provider_payment_id": event.Object.ID,
		"payment_status":      paymentStatus,
		"order_id":            orderID,
		"product_id":          event.Object.ProductID,
	}

	return payment.NormalizedWebhook{
		ProviderEventID:   event.ID,
		EventType:         event.EventType,
		BusinessEventType: businessEventType,
		ProviderPaymentID: event.Object.ID,
		PaymentStatus:     paymentStatus,
		OrderID:           orderID,
		SafeSnapshot:      snapshot,
	}, nil
}

func validateCreateConfig(cfg payment.ProviderConfig) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return providererror.New(providererror.CategoryValidation, false, "payment base URL is required", nil)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return providererror.New(providererror.CategoryAuth, false, "payment credential is required", nil)
	}
	if strings.TrimSpace(cfg.ProductID) == "" {
		return providererror.New(providererror.CategoryValidation, false, "payment product is required", nil)
	}
	return nil
}

func verifySignature(secret string, rawPayload []byte, signature string) error {
	secret = strings.TrimSpace(secret)
	signature = strings.TrimSpace(signature)
	if secret == "" {
		return providererror.New(providererror.CategoryAuth, false, "payment webhook secret is required", nil)
	}
	if signature == "" {
		return providererror.New(providererror.CategoryAuth, false, "payment webhook signature is required", nil)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(rawPayload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
		return providererror.New(providererror.CategoryAuth, false, "payment webhook signature is invalid", nil)
	}
	return nil
}

func checkoutURL(base string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("base URL must include scheme and host")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/checkouts"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func customerFromRequest(customer payment.Customer) *checkoutCustomer {
	email := strings.TrimSpace(customer.Email)
	if email == "" {
		return nil
	}
	return &checkoutCustomer{Email: email}
}

func metadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func providerErrorFromStatus(statusCode int) *providererror.Error {
	category := providererror.CategoryProviderInternal
	retryable := statusCode >= 500 || statusCode == http.StatusTooManyRequests || statusCode == http.StatusRequestTimeout

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		category = providererror.CategoryAuth
		retryable = false
	case http.StatusTooManyRequests:
		category = providererror.CategoryRateLimit
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		category = providererror.CategoryTimeout
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		category = providererror.CategoryValidation
		retryable = false
	default:
		if statusCode >= 400 && statusCode < 500 {
			category = providererror.CategoryPermanent
			retryable = false
		}
	}

	return &providererror.Error{
		Category:    category,
		Retryable:   retryable,
		SafeMessage: "payment provider request failed",
	}
}
