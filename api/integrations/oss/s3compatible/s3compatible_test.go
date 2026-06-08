package s3compatible

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tfnick/go-svelte-starter/api/framework/integrations/providererror"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/oss"
)

type fakeHTTPDoer struct {
	req  *http.Request
	body string
	resp *http.Response
	err  error
}

func (d *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	d.req = req
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		d.body = string(body)
	}
	if d.err != nil {
		return nil, d.err
	}
	if d.resp != nil {
		return d.resp, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Etag": []string{`"etag-1"`}, "X-Amz-Request-Id": []string{"req-1"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func TestPutObjectSignsS3CompatibleRequest(t *testing.T) {
	doer := &fakeHTTPDoer{}
	adapter := NewAdapter(doer)
	adapter.now = func() time.Time {
		return time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	}

	result, err := adapter.PutObject(t.Context(), testProviderConfig(), oss.PutObjectRequest{
		Key:         "settings/site-logo.png",
		Body:        strings.NewReader("logo"),
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	if doer.req.Method != http.MethodPut {
		t.Fatalf("expected PUT method, got %s", doer.req.Method)
	}
	if doer.req.URL.String() != "https://r2.example.com/assets/public/settings/site-logo.png" {
		t.Fatalf("unexpected request URL: %s", doer.req.URL.String())
	}
	if doer.body != "logo" {
		t.Fatalf("unexpected request body: %q", doer.body)
	}
	if doer.req.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("expected content type header, got %q", doer.req.Header.Get("Content-Type"))
	}
	if doer.req.Header.Get("X-Amz-Date") != "20260608T120000Z" {
		t.Fatalf("unexpected x-amz-date: %s", doer.req.Header.Get("X-Amz-Date"))
	}
	if !strings.HasPrefix(doer.req.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=ak/20260608/auto/s3/aws4_request") {
		t.Fatalf("missing AWS authorization header: %s", doer.req.Header.Get("Authorization"))
	}
	if result.Key != "settings/site-logo.png" || result.Size != 4 || result.ETag != "etag-1" || result.ProviderRequestID != "req-1" {
		t.Fatalf("unexpected put result: %#v", result)
	}
	if result.PublicURL != "https://assets.example.com/public/settings/site-logo.png" {
		t.Fatalf("unexpected public URL: %q", result.PublicURL)
	}
}

func TestGetObjectReturnsProviderBody(t *testing.T) {
	doer := &fakeHTTPDoer{
		resp: &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": []string{"image/png"}, "Etag": []string{`"etag-get"`}},
			Body:          io.NopCloser(strings.NewReader("image-bytes")),
			ContentLength: int64(len("image-bytes")),
		},
	}
	adapter := NewAdapter(doer)
	adapter.now = func() time.Time {
		return time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	}

	result, err := adapter.GetObject(t.Context(), testProviderConfig(), oss.GetObjectRequest{Key: "settings/site-logo.png"})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read result body: %v", err)
	}

	if doer.req.Method != http.MethodGet {
		t.Fatalf("expected GET method, got %s", doer.req.Method)
	}
	if string(body) != "image-bytes" || result.ContentType != "image/png" || result.Size != int64(len("image-bytes")) {
		t.Fatalf("unexpected get result: body=%q result=%#v", string(body), result)
	}
}

func TestGetObjectMapsNotFoundToPermanentProviderError(t *testing.T) {
	doer := &fakeHTTPDoer{
		resp: &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"X-Amz-Request-Id": []string{"missing-req"}},
			Body:       io.NopCloser(strings.NewReader("not found")),
		},
	}
	adapter := NewAdapter(doer)

	_, err := adapter.GetObject(t.Context(), testProviderConfig(), oss.GetObjectRequest{Key: "missing.png"})
	if err == nil {
		t.Fatalf("expected missing object error")
	}
	providerErr, ok := providererror.From(err)
	if !ok {
		t.Fatalf("expected provider error, got %T %v", err, err)
	}
	if providerErr.Category != providererror.CategoryPermanent || providerErr.Retryable || providerErr.ProviderRequestID != "missing-req" {
		t.Fatalf("unexpected provider error: %#v", providerErr)
	}
}

func testProviderConfig() oss.ProviderConfig {
	return oss.ProviderConfig{
		EndpointURL:     "https://r2.example.com",
		Bucket:          "assets",
		Region:          "auto",
		PublicBaseURL:   "https://assets.example.com",
		KeyPrefix:       "public",
		AccessKeyID:     "ak",
		SecretAccessKey: "sk",
	}
}
