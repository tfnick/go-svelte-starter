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
	BaseURL string `json:"base_url"`
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
	if strings.TrimSpace(channelConfig.BaseURL) == "" {
		return embedding.ProviderConfig{}, fmt.Errorf("channel base_url is required")
	}

	apiKey := config.Credential.ValueText
	if strings.TrimSpace(apiKey) == "" {
		return embedding.ProviderConfig{}, fmt.Errorf("credential is empty")
	}

	modelSettings := map[string]interface{}{}
	if strings.TrimSpace(config.Model.DefaultParamsJSON) != "" {
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(config.Model.DefaultParamsJSON), &params); err == nil {
			modelSettings = params
		}
	}

	providerSettings := map[string]interface{}{}
	if strings.TrimSpace(config.Channel.MetadataJSON) != "" {
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(config.Channel.MetadataJSON), &meta); err == nil {
			providerSettings = meta
		}
	}

	return embedding.ProviderConfig{
		ChannelID:       config.Channel.ID,
		ChannelCode:     config.Channel.ChannelCode,
		AdapterKey:      config.Channel.AdapterKey,
		Provider:        config.Channel.ProviderCode,
		CredentialValue:  apiKey,
		BaseURL:          channelConfig.BaseURL,
		ModelCode:        config.Model.ModelCode,
		ProviderModelID:  config.Model.ProviderModelID,
		ModelSettings:    modelSettings,
		ProviderSettings: providerSettings,
	}, nil
}
