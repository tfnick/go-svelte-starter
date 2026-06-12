package usecase_test

import (
	"context"
	"strings"
	"testing"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/embedding"
	"github.com/tfnick/sqlx"
)

type fakeEmbeddingAdapter struct {
	t *testing.T
}

func (a fakeEmbeddingAdapter) Embed(_ context.Context, cfg embedding.ProviderConfig, req embedding.EmbedRequest) (embedding.EmbedResult, error) {
	a.t.Helper()
	if cfg.ChannelCode != "kb-embedding-channel-only" {
		a.t.Fatalf("unexpected embedding channel code: %s", cfg.ChannelCode)
	}
	if cfg.ModelCode != "deepseek-embedding" || cfg.ProviderModelID != "deepseek-embedding" {
		a.t.Fatalf("unexpected embedding model mapping: %#v", cfg)
	}
	dimensions, ok := req.Params["dimensions"].(float64)
	if !ok || dimensions != 64 {
		a.t.Fatalf("expected dimensions=64 from model defaults, got %#v", req.Params)
	}
	if len(req.Texts) == 0 {
		a.t.Fatalf("expected chunk texts in embedding request")
	}

	vectors := make([]embedding.Vector, len(req.Texts))
	for i := range req.Texts {
		values := make([]float32, 64)
		values[i%len(values)] = 1
		vectors[i] = embedding.Vector{Values: values}
	}

	return embedding.EmbedResult{
		Vectors:         vectors,
		ModelCode:       cfg.ModelCode,
		ProviderModelID: cfg.ProviderModelID,
		Dimensions:      64,
		Usage:           embedding.Usage{PromptTokens: len(req.Texts), TotalTokens: len(req.Texts)},
	}, nil
}

func TestIndexDocumentUsesEmbeddingChannelOnlyConfig(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedEmbeddingChannelOnlyConfig(t, appDB, "kb-embedding-channel-only", "kb-embedding-api-key")

	if err := usecase.RegisterEmbeddingAdapter("embedding.deepseek.openai_compatible", fakeEmbeddingAdapter{t: t}); err != nil {
		t.Fatalf("register fake embedding adapter: %v", err)
	}

	source, err := models.CreateKnowledgeSource(t.Context(), models.SaveKnowledgeSourceCmd{
		ID:         "kb-embedding-source",
		Title:      "Embedding Source",
		SourceType: models.KBSourceTypeManual,
		Enabled:    true,
		Content:    "Embedding configuration should work after only creating a channel in Parameter.",
	})
	if err != nil {
		t.Fatalf("create knowledge source: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	if err := usecase.IndexDocument(ctx, usecase.IndexDocumentCmd{DocumentID: source.DocumentID}); err != nil {
		t.Fatalf("index document: %v", err)
	}

	doc, err := models.GetKBDocumentByID(t.Context(), source.DocumentID)
	if err != nil {
		t.Fatalf("load indexed document: %v", err)
	}
	if doc.IndexStatus != models.KBIndexStatusIndexed || doc.LastIndexError != "" {
		t.Fatalf("expected indexed document without error, got %#v", doc)
	}

	var chunk struct {
		EmbeddingModelCode       string `db:"embedding_model_code"`
		EmbeddingProviderModelID string `db:"embedding_provider_model_id"`
		EmbeddingDimensions      int    `db:"embedding_dimensions"`
	}
	if err := appDB.Get(&chunk, `
		SELECT embedding_model_code, embedding_provider_model_id, embedding_dimensions
		FROM kb_chunks
		WHERE document_id = ?
		LIMIT 1`, source.DocumentID); err != nil {
		t.Fatalf("load chunk: %v", err)
	}
	if chunk.EmbeddingModelCode != "deepseek-embedding" || chunk.EmbeddingProviderModelID != "deepseek-embedding" || chunk.EmbeddingDimensions != 64 {
		t.Fatalf("unexpected chunk embedding metadata: %#v", chunk)
	}
}

func TestIndexDocumentMissingEmbeddingConfigMentionsParameterMenu(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	source, err := models.CreateKnowledgeSource(t.Context(), models.SaveKnowledgeSourceCmd{
		ID:         "kb-missing-embedding-source",
		Title:      "Missing Embedding Source",
		SourceType: models.KBSourceTypeManual,
		Enabled:    true,
		Content:    "This document should fail because no embedding channel exists.",
	})
	if err != nil {
		t.Fatalf("create knowledge source: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	if err := usecase.IndexDocument(ctx, usecase.IndexDocumentCmd{DocumentID: source.DocumentID}); err == nil {
		t.Fatalf("expected missing embedding config error")
	}

	doc, err := models.GetKBDocumentByID(t.Context(), source.DocumentID)
	if err != nil {
		t.Fatalf("load failed document: %v", err)
	}
	if doc.IndexStatus != models.KBIndexStatusFailed {
		t.Fatalf("expected failed index status, got %#v", doc)
	}
	if !strings.Contains(doc.LastIndexError, "Parameter > Embedding") {
		t.Fatalf("expected Parameter > Embedding guidance, got %q", doc.LastIndexError)
	}
	if strings.Contains(doc.LastIndexError, "Settings > Integrations") {
		t.Fatalf("did not expect stale settings guidance, got %q", doc.LastIndexError)
	}
}

func seedEmbeddingChannelOnlyConfig(t *testing.T, appDB *sqlx.DB, channelCode string, apiKey string) {
	t.Helper()

	credentialValue, err := credentialsForTest(apiKey)
	if err != nil {
		t.Fatalf("prepare credential: %v", err)
	}
	credentialID := channelCode + "-credential"
	channelID := channelCode + "-channel"

	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO integration_credentials (id, credential_type, ciphertext, key_version, masked_value, value_text, enabled)
		VALUES (?, 'api_key', ?, '', '', ?, 1)
	`), credentialID, credentialValue, credentialValue); err != nil {
		t.Fatalf("insert embedding credential: %v", err)
	}
	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO integration_channels (
			id, scenario, channel_code, provider_code, adapter_key, environment, enabled, priority, credential_id, config_json
		) VALUES (?, 'embedding', ?, 'deepseek', 'embedding.deepseek.openai_compatible', 'test', 1, 1, ?, '{"base_url":"https://api.deepseek.com"}')
	`), channelID, channelCode, credentialID); err != nil {
		t.Fatalf("insert embedding channel: %v", err)
	}
}
