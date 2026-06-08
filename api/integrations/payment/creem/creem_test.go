package creem

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tfnick/go-svelte-starter/api/framework/integrations/providererror"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/payment"
)

type fakeHTTPDoer struct {
	t  *testing.T
	do func(*http.Request) (*http.Response, error)
}

func (d fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	d.t.Helper()
	return d.do(req)
}

func TestCreatePaymentBuildsCreemCheckoutRequest(t *testing.T) {
	adapter := NewAdapter(fakeHTTPDoer{
		t: t,
		do: func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.URL.String() != "https://test-api.creem.io/v1/checkouts" {
				t.Fatalf("unexpected request target: %s %s", req.Method, req.URL.String())
			}
			if req.Header.Get("x-api-key") != "test-api-key" {
				t.Fatalf("missing api key header")
			}
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			text := string(body)
			for _, expected := range []string{
				`"product_id":"prod_123"`,
				`"request_id":"order-1"`,
				`"success_url":"https://example.com/success"`,
				`"email":"ada@example.com"`,
				`"order_id":"order-1"`,
			} {
				if !strings.Contains(text, expected) {
					t.Fatalf("expected body to contain %s, got %s", expected, text)
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"ch_123","checkout_url":"https://checkout.creem.io/ch_123","product_id":"prod_123","status":"pending"}`)),
				Header:     http.Header{},
			}, nil
		},
	})

	result, err := adapter.CreatePayment(context.Background(), payment.ProviderConfig{
		BaseURL:   "https://test-api.creem.io/v1",
		APIKey:    "test-api-key",
		ProductID: "prod_123",
	}, payment.CreatePaymentRequest{
		OrderID: "order-1",
		Customer: payment.Customer{
			Email: "ada@example.com",
		},
		Metadata:   map[string]string{"order_id": "order-1"},
		SuccessURL: "https://example.com/success",
	})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if result.ProviderPaymentID != "ch_123" || result.CheckoutURL == "" || result.ProviderRequestID != "ch_123" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestNormalizePaymentWebhookVerifiesSignatureAndMapsOrderID(t *testing.T) {
	payload := []byte(`{"id":"evt_1","eventType":"checkout.completed","object":{"id":"ch_1","request_id":"order-1","status":"completed","product_id":"prod_1","metadata":{"order_id":"order-1"}}}`)
	signature := signPayload("webhook-secret", payload)

	adapter := NewAdapter(nil)
	normalized, err := adapter.NormalizePaymentWebhook(context.Background(), payment.ProviderConfig{
		WebhookSecret: "webhook-secret",
	}, payment.WebhookRequest{
		RawPayload:      payload,
		Signature:       signature,
		VerifySignature: true,
	})
	if err != nil {
		t.Fatalf("normalize webhook: %v", err)
	}
	if normalized.ProviderEventID != "evt_1" || normalized.ProviderPaymentID != "ch_1" {
		t.Fatalf("unexpected provider IDs: %#v", normalized)
	}
	if normalized.BusinessEventType != payment.WebhookEventPaymentSucceeded || normalized.OrderID != "order-1" {
		t.Fatalf("unexpected business mapping: %#v", normalized)
	}
}

func TestNormalizePaymentWebhookMapsNestedCreemCheckoutPayload(t *testing.T) {
	payload := []byte(`{"id":"evt_nested","eventType":"checkout.completed","object":{"id":"ch_nested","object":"checkout","request_id":"order-nested","order":{"id":"ord_provider","product":"prod_from_order","status":"paid"},"product":{"id":"prod_from_product"},"metadata":{"user_id":"user-1"}}}`)
	signature := signPayload("webhook-secret", payload)

	adapter := NewAdapter(nil)
	normalized, err := adapter.NormalizePaymentWebhook(context.Background(), payment.ProviderConfig{
		WebhookSecret: "webhook-secret",
	}, payment.WebhookRequest{
		RawPayload:      payload,
		Signature:       strings.ToUpper(signature[:16]) + " \n " + strings.ToUpper(signature[16:]),
		VerifySignature: true,
	})
	if err != nil {
		t.Fatalf("normalize webhook: %v", err)
	}
	if normalized.OrderID != "order-nested" {
		t.Fatalf("expected request_id to map to order ID, got %#v", normalized)
	}
	if normalized.PaymentStatus != "succeeded" || normalized.BusinessEventType != payment.WebhookEventPaymentSucceeded {
		t.Fatalf("unexpected payment status mapping: %#v", normalized)
	}
	if normalized.SafeSnapshot["product_id"] != "prod_from_product" {
		t.Fatalf("expected nested product ID in snapshot, got %#v", normalized.SafeSnapshot)
	}
}

func TestNormalizePaymentWebhookRejectsInvalidSignature(t *testing.T) {
	adapter := NewAdapter(nil)
	_, err := adapter.NormalizePaymentWebhook(context.Background(), payment.ProviderConfig{
		WebhookSecret: "webhook-secret",
	}, payment.WebhookRequest{
		RawPayload:      []byte(`{"id":"evt_1"}`),
		Signature:       "bad",
		VerifySignature: true,
	})
	if err == nil {
		t.Fatal("expected signature error")
	}
}

func TestNormalizePaymentWebhookRequiresSignature(t *testing.T) {
	adapter := NewAdapter(nil)
	_, err := adapter.NormalizePaymentWebhook(context.Background(), payment.ProviderConfig{
		WebhookSecret: "webhook-secret",
	}, payment.WebhookRequest{
		RawPayload:      []byte(`{"id":"evt_1"}`),
		VerifySignature: true,
	})
	if err == nil {
		t.Fatal("expected missing signature error")
	}
	providerErr, ok := providererror.From(err)
	if !ok || providerErr.Category != providererror.CategoryAuth {
		t.Fatalf("expected auth provider error, got %v", err)
	}
}

func signPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
