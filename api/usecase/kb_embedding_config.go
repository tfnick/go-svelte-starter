package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/embedding"
)

const embeddingOperationCreate = embedding.OperationCreate

type embeddingChannelConfig struct {
	BaseURL        string `json:"base_url"`
	APIStyle       string `json:"api_style"`
	EndpointPath   string `json:"endpoint_path"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format"`
	Dimensions     int    `json:"dimensions"`
}

// embeddingProviderConfig converts an IntegrationEmbeddingConfig into an embedding.ProviderConfig,
// following the same pattern as llmProviderConfig in llm_summary.go.
func embeddingProviderConfig(config models.IntegrationEmbeddingConfig) (embedding.ProviderConfig, error) {
	channelConfig := embeddingChannelConfig{}
	if strings.TrimSpace(config.Channel.ConfigJSON) != "" {
		if err := json.Unmarshal([]byte(config.Channel.ConfigJSON), &channelConfig); err != nil {
			return embedding.ProviderConfig{}, fmt.Errorf("parse channel config failed: %w", err)
		}
	}
	if strings.TrimSpace(channelConfig.BaseURL) == "" && !isLocalHashEmbeddingAdapter(config.Channel.AdapterKey) {
		return embedding.ProviderConfig{}, fmt.Errorf("channel base_url is required")
	}

	apiKey := config.Credential.ValueText
	if strings.TrimSpace(apiKey) == "" && !isLocalHashEmbeddingAdapter(config.Channel.AdapterKey) {
		return embedding.ProviderConfig{}, fmt.Errorf("credential is empty")
	}

	modelSettings := map[string]interface{}{}
	if strings.TrimSpace(config.Model.DefaultParamsJSON) != "" {
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(config.Model.DefaultParamsJSON), &params); err == nil {
			modelSettings = params
		}
	}
	if dimensions := channelConfig.Dimensions; dimensions > 0 {
		modelSettings["dimensions"] = dimensions
	}
	if encodingFormat := strings.TrimSpace(channelConfig.EncodingFormat); encodingFormat != "" {
		modelSettings["encoding_format"] = encodingFormat
	}

	providerSettings := map[string]interface{}{}
	if strings.TrimSpace(config.Channel.MetadataJSON) != "" {
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(config.Channel.MetadataJSON), &meta); err == nil {
			providerSettings = meta
		}
	}
	if apiStyle := strings.TrimSpace(channelConfig.APIStyle); apiStyle != "" {
		providerSettings["api_style"] = apiStyle
	}
	if endpointPath := strings.TrimSpace(channelConfig.EndpointPath); endpointPath != "" {
		providerSettings["endpoint_path"] = endpointPath
	}

	modelCode := config.Model.ModelCode
	providerModelID := config.Model.ProviderModelID
	if model := strings.TrimSpace(channelConfig.Model); model != "" {
		providerModelID = model
		modelCode = embeddingModelCodeForProviderModel(model, modelCode)
	}

	return embedding.ProviderConfig{
		ChannelID:        config.Channel.ID,
		ChannelCode:      config.Channel.ChannelCode,
		AdapterKey:       config.Channel.AdapterKey,
		Provider:         config.Channel.ProviderCode,
		CredentialValue:  apiKey,
		BaseURL:          channelConfig.BaseURL,
		ModelCode:        modelCode,
		ProviderModelID:  providerModelID,
		ModelSettings:    modelSettings,
		ProviderSettings: providerSettings,
	}, nil
}

func isLocalHashEmbeddingAdapter(adapterKey string) bool {
	normalized := strings.TrimSpace(adapterKey)
	return normalized == "embedding.local_hash_64" || strings.HasPrefix(normalized, "embedding.local_hash_64.")
}

func embeddingModelCodeForProviderModel(providerModelID string, fallback string) string {
	switch strings.TrimSpace(providerModelID) {
	case "Qwen/Qwen3-Embedding-0.6B":
		return "qwen3-embedding-0.6b"
	case "Qwen/Qwen3-Embedding-4B":
		return "qwen3-embedding-4b"
	case "Qwen/Qwen3-Embedding-8B":
		return "qwen3-embedding-8b"
	default:
		return fallback
	}
}
