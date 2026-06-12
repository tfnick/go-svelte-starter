package usecase_test

import (
	"context"
	"strings"
	"testing"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/providers/embedding/localhash"
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

func TestIndexDocumentUsesDefaultLocalHashEmbeddingConfig(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	adapterKey := "embedding.local_hash_64.test.default"
	if err := usecase.RegisterEmbeddingAdapter(adapterKey, localhash.NewAdapter(64)); err != nil {
		t.Fatalf("register local embedding adapter: %v", err)
	}
	if _, err := appDB.Exec(`
		UPDATE integration_channels
		SET adapter_key = ?
		WHERE scenario = 'embedding' AND channel_code = 'local-hash-64'
	`, adapterKey); err != nil {
		t.Fatalf("isolate local embedding adapter key: %v", err)
	}

	source, err := models.CreateKnowledgeSource(t.Context(), models.SaveKnowledgeSourceCmd{
		ID:         "kb-default-local-embedding-source",
		Title:      "Default Local Embedding Source",
		SourceType: models.KBSourceTypeManual,
		Enabled:    true,
		Content:    "Default local hash embedding should index without calling an external provider.",
	})
	if err != nil {
		t.Fatalf("create knowledge source: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	if err := usecase.IndexDocument(ctx, usecase.IndexDocumentCmd{DocumentID: source.DocumentID}); err != nil {
		doc, loadErr := models.GetKBDocumentByID(t.Context(), source.DocumentID)
		if loadErr != nil {
			t.Fatalf("index document: %v; load failed document: %v", err, loadErr)
		}
		t.Fatalf("index document: %v; last index error: %s", err, doc.LastIndexError)
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
	if chunk.EmbeddingModelCode != "local-hash-64" || chunk.EmbeddingProviderModelID != "local-hash-64" || chunk.EmbeddingDimensions != 64 {
		t.Fatalf("unexpected chunk embedding metadata: %#v", chunk)
	}
}

func TestIndexDocumentMissingEmbeddingConfigMentionsParameterMenu(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	for _, query := range []string{
		`DELETE FROM integration_operation_configs WHERE scenario = 'embedding'`,
		`DELETE FROM integration_model_options WHERE scenario = 'embedding'`,
		`DELETE FROM integration_channels WHERE scenario = 'embedding'`,
		`DELETE FROM integration_credentials WHERE credential_type = 'none'`,
	} {
		if _, err := appDB.Exec(query); err != nil {
			t.Fatalf("remove default embedding config: %v", err)
		}
	}

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

func TestIndexDocumentWithoutContentReturnsValidation(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	source, err := models.CreateKnowledgeSource(t.Context(), models.SaveKnowledgeSourceCmd{
		ID:         "kb-empty-content-source",
		Title:      "Empty Content Source",
		SourceType: models.KBSourceTypeManual,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create knowledge source: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	err = usecase.IndexDocument(ctx, usecase.IndexDocumentCmd{DocumentID: source.DocumentID})
	if err == nil {
		t.Fatalf("expected empty content validation error")
	}
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation code, got %q: %v", fwusecase.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "document content is required before indexing") {
		t.Fatalf("unexpected safe error message: %v", err)
	}

	doc, err := models.GetKBDocumentByID(t.Context(), source.DocumentID)
	if err != nil {
		t.Fatalf("load failed document: %v", err)
	}
	if doc.IndexStatus != models.KBIndexStatusFailed || doc.LastIndexError != "document has no content" {
		t.Fatalf("expected failed status with content error, got %#v", doc)
	}

	sources, err := usecase.ListKBSources(ctx)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 || sources[0].IndexStatus != models.KBIndexStatusFailed || sources[0].LastIndexError != "document has no content" {
		t.Fatalf("expected source status to mirror document failure, got %#v", sources)
	}
}

func TestIndexDocumentWithoutContentDoesNotDeleteExistingChunks(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	source, err := models.CreateKnowledgeSource(t.Context(), models.SaveKnowledgeSourceCmd{
		ID:          "kb-empty-content-keeps-chunks-source",
		Title:       "Empty Content Keeps Chunks Source",
		SourceType:  models.KBSourceTypeManual,
		Enabled:     true,
		ContentHash: "old-hash",
	})
	if err != nil {
		t.Fatalf("create knowledge source: %v", err)
	}
	if _, err := appDB.Exec(`
		INSERT INTO kb_chunks (
		  id, source_id, document_id, chunk_index, content, content_hash, token_count, char_count,
		  embedding_model_code, embedding_provider_model_id, embedding_dimensions, embedding_status,
		  enabled, created_at, updated_at
		) VALUES (
		  'old-chunk', ?, ?, 0, 'old indexed content', 'old-chunk-hash', 3, 19,
		  'old-model', 'old-provider-model', 64, 'embedded', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`, source.ID, source.DocumentID); err != nil {
		t.Fatalf("insert old chunk: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	err = usecase.IndexDocument(ctx, usecase.IndexDocumentCmd{DocumentID: source.DocumentID})
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation code, got %q: %v", fwusecase.CodeOf(err), err)
	}

	var chunkCount int
	if err := appDB.Get(&chunkCount, `SELECT COUNT(1) FROM kb_chunks WHERE document_id = ?`, source.DocumentID); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if chunkCount != 1 {
		t.Fatalf("expected empty-content reindex to preserve existing chunks, got %d", chunkCount)
	}
}

func TestCreateKBDocumentUsesRequestedSource(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	source, err := models.CreateKnowledgeSource(t.Context(), models.SaveKnowledgeSourceCmd{
		ID:         "kb-document-parent",
		Title:      "Parent Source",
		SourceType: models.KBSourceTypeManual,
		Enabled:    true,
		Content:    "Parent source description",
	})
	if err != nil {
		t.Fatalf("create knowledge source: %v", err)
	}

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	doc, err := usecase.CreateKBDocument(ctx, usecase.CreateKBDocumentCmd{
		SourceID: source.ID,
		Title:    "Child Document",
		Content:  "Child document content",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if doc.SourceID != source.ID {
		t.Fatalf("expected document source %q, got %#v", source.ID, doc)
	}

	sources, err := usecase.ListKBSources(ctx)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected one source after creating child document, got %d: %#v", len(sources), sources)
	}
	if sources[0].Description != "Parent source description" {
		t.Fatalf("expected source description preserved, got %#v", sources[0])
	}

	docs, err := usecase.ListKBDocuments(ctx, source.ID)
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected parent document plus child document, got %d: %#v", len(docs), docs)
	}
	foundChild := false
	for _, candidate := range docs {
		if candidate.ID == doc.ID && candidate.SourceID == source.ID && candidate.Content == "Child document content" {
			foundChild = true
		}
	}
	if !foundChild {
		t.Fatalf("created document not found under requested source: %#v", docs)
	}
}

func TestUpdateKBSourcePreservesDescriptionForFrontendRoundTrip(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)

	source, err := usecase.CreateKBSource(ctx, usecase.CreateKBSourceCmd{
		Title:       "Original Source",
		Description: "Original description",
		SourceType:  models.KBSourceTypeManual,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if source.Description != "Original description" {
		t.Fatalf("expected created description, got %#v", source)
	}

	updated, err := usecase.UpdateKBSource(ctx, usecase.UpdateKBSourceCmd{
		ID:          source.ID,
		Title:       "Updated Source",
		Description: source.Description,
	})
	if err != nil {
		t.Fatalf("update source: %v", err)
	}
	if updated.Description != "Original description" {
		t.Fatalf("expected description to round-trip, got %#v", updated)
	}

	docs, err := usecase.ListKBDocuments(ctx, source.ID)
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(docs) != 1 || docs[0].Content != "Original description" {
		t.Fatalf("source update should preserve primary document content, got %#v", docs)
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
	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO integration_operation_configs (
			id, scenario, operation, channel_code, model_code, enabled, config_json
		) VALUES (?, 'embedding', 'embedding_create', ?, '', 1, '{}')
		ON CONFLICT(scenario, operation) DO UPDATE SET
			channel_code = excluded.channel_code,
			model_code = excluded.model_code,
			enabled = excluded.enabled,
			config_json = excluded.config_json
	`), channelCode+"-operation", channelCode); err != nil {
		t.Fatalf("insert embedding operation config: %v", err)
	}
}
