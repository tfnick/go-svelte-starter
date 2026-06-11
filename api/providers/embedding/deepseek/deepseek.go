package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tfnick/go-svelte-starter/api/framework/integrations/providererror"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/embedding"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Adapter struct {
	client HTTPDoer
}

func NewAdapter(client HTTPDoer) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Adapter{client: client}
}

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embedData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

type embedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type embedResponse struct {
	Object string      `json:"object"`
	Data   []embedData `json:"data"`
	Model  string      `json:"model"`
	Usage  embedUsage  `json:"usage"`
}

func (a *Adapter) Embed(ctx context.Context, cfg embedding.ProviderConfig, req embedding.EmbedRequest) (embedding.EmbedResult, error) {
	if err := validateConfig(cfg); err != nil {
		return embedding.EmbedResult{}, err
	}
	if len(req.Texts) == 0 {
		return embedding.EmbedResult{}, providererror.New(providererror.CategoryValidation, false, "embedding texts are required", nil)
	}

	body, err := buildEmbedRequestBody(cfg, req)
	if err != nil {
		return embedding.EmbedResult{}, err
	}

	endpoint, err := embeddingsURL(cfg.BaseURL)
	if err != nil {
		return embedding.EmbedResult{}, providererror.New(providererror.CategoryValidation, false, "embedding base URL is invalid", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return embedding.EmbedResult{}, providererror.New(providererror.CategoryTemporary, true, "failed to create embedding request", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+cfg.CredentialValue)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return embedding.EmbedResult{}, providererror.New(providererror.CategoryTemporary, true, "embedding provider request failed", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<24))
	if err != nil {
		return embedding.EmbedResult{}, providererror.New(providererror.CategoryTemporary, true, "failed to read embedding response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return embedding.EmbedResult{}, embeddingErrorFromStatus(resp.StatusCode, firstHeader(resp.Header, "X-Request-ID"))
	}

	var parsed embedResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return embedding.EmbedResult{}, providererror.New(providererror.CategoryProviderInternal, true, "embedding response is invalid", err)
	}
	if len(parsed.Data) == 0 {
		return embedding.EmbedResult{}, providererror.New(providererror.CategoryProviderInternal, true, "embedding response is empty", nil)
	}

	vectors := make([]embedding.Vector, len(parsed.Data))
	for i, data := range parsed.Data {
		vectors[i] = embedding.Vector{Values: data.Embedding}
	}

	providerRequestID := firstHeader(resp.Header, "X-Request-ID", "X-Request-Id")
	modelCode := strings.TrimSpace(cfg.ModelCode)
	if modelCode == "" {
		modelCode = "deepseek-embedding"
	}

	return embedding.EmbedResult{
		Vectors:           vectors,
		ProviderRequestID: providerRequestID,
		ModelCode:         modelCode,
		ProviderModelID:   strings.TrimSpace(cfg.ProviderModelID),
		Dimensions:        len(vectors[0].Values),
		Usage:             embedding.Usage{PromptTokens: parsed.Usage.PromptTokens, TotalTokens: parsed.Usage.TotalTokens},
	}, nil
}

func validateConfig(cfg embedding.ProviderConfig) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return providererror.New(providererror.CategoryValidation, false, "embedding base URL is required", nil)
	}
	if strings.TrimSpace(cfg.CredentialValue) == "" {
		return providererror.New(providererror.CategoryAuth, false, "embedding credential is required", nil)
	}
	if strings.TrimSpace(cfg.ProviderModelID) == "" {
		return providererror.New(providererror.CategoryValidation, false, "embedding model is required", nil)
	}
	return nil
}

func buildEmbedRequestBody(cfg embedding.ProviderConfig, req embedding.EmbedRequest) ([]byte, error) {
	payload := embedRequest{
		Model: cfg.ProviderModelID,
		Input: req.Texts,
	}
	if dim, ok := req.Params["dimensions"]; ok {
		if d, ok := dim.(float64); ok && d > 0 {
			payload.Dimensions = int(d)
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, providererror.New(providererror.CategoryValidation, false, "embedding request is invalid", err)
	}
	return body, nil
}

func embeddingsURL(base string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("base URL must include scheme and host")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/embeddings"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func embeddingErrorFromStatus(statusCode int, providerRequestID string) *providererror.Error {
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
		Category:          category,
		Retryable:         retryable,
		SafeMessage:       "embedding provider request failed",
		ProviderRequestID: providerRequestID,
	}
}

func firstHeader(header http.Header, names ...string) string {
	for _, name := range names {
		if value := header.Get(name); value != "" {
			return value
		}
	}
	return ""
}
