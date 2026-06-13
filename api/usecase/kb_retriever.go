package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/embedding"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/kb"
)

var (
	errKBDocumentHasNoContent = errors.New("document has no content")
	errKBDocumentHasNoChunks  = errors.New("document produced no chunks")
)

// SQLiteVecRetriever implements kb.Retriever and kb.Indexer using the app SQLite database
// and the sqlite-vec vec0 virtual table for KNN vector search.
type SQLiteVecRetriever struct{}

// NewSQLiteVecRetriever creates a new retriever backed by the default SQLite database manager.
func NewSQLiteVecRetriever() *SQLiteVecRetriever {
	return &SQLiteVecRetriever{}
}

// Search performs a KNN vector search against the kb_chunk_embedding_vec table,
// joins with kb_chunks to retrieve content and metadata, and filters by enabled sources/documents/chunks.
func (r *SQLiteVecRetriever) Search(ctx context.Context, queryEmbedding []float32, topK int, minScore float64) ([]kb.Chunk, error) {
	if topK <= 0 {
		topK = 5
	}

	embeddingJSON, err := json.Marshal(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("marshal query embedding: %w", err)
	}

	results, err := models.SearchKnowledgeChunks(ctx, string(embeddingJSON), topK)
	if err != nil {
		return nil, fmt.Errorf("search knowledge chunks: %w", err)
	}

	chunks := make([]kb.Chunk, 0, len(results))
	for _, result := range results {
		if minScore > 0 && result.Distance > minScore {
			continue
		}
		chunks = append(chunks, kb.Chunk{
			ChunkID:  result.ChunkID,
			SourceID: result.SourceID,
			Content:  result.Content,
			Score:    result.Distance,
		})
	}

	if len(chunks) == 0 && len(results) > 0 {
		return chunks, nil
	}

	return chunks, nil
}

// IndexDocument chunks a document's content, generates embeddings, and stores them.
// It uses the DefaultEmbeddingProvider config pattern, following the LLM config loading approach.
func (r *SQLiteVecRetriever) IndexDocument(ctx context.Context, documentID string) error {
	// Delegate to the full IndexDocument usecase flow.
	// We create a fwusecase context from the provided context and call IndexDocument.
	// This ensures the full lifecycle (status updates, chunking, embedding, storing) is executed.
	return indexDocumentInternal(ctx, documentID)
}

// indexDocumentInternal is the internal implementation that performs the actual indexing.
// It is called by both the async task handler and the synchronous fallback.
func indexDocumentInternal(ctx context.Context, documentID string) error {
	doc, err := models.GetKBDocumentByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("load document for indexing: %w", err)
	}

	// Set status to processing
	if err := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusProcessing, ""); err != nil {
		return fmt.Errorf("set document status to processing: %w", err)
	}

	// Determine content to index
	content := doc.Content
	if content == "" {
		content = doc.ExtractedText
	}
	if content == "" {
		if err := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed, "document has no content"); err != nil {
			return fmt.Errorf("set document status to failed: %w", err)
		}
		return fmt.Errorf("document %s has no content: %w", documentID, errKBDocumentHasNoContent)
	}

	// Chunk the content
	chunker := &SimpleChunker{MaxTokens: 500}
	chunks := chunker.Chunk(content)
	if len(chunks) == 0 {
		if err := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed, "no chunks produced from document"); err != nil {
			return fmt.Errorf("set document status to failed: %w", err)
		}
		return fmt.Errorf("document %s produced no chunks: %w", documentID, errKBDocumentHasNoChunks)
	}

	// Load embedding config
	embedCfg, err := models.GetEnabledEmbeddingConfig(ctx, models.EmbeddingConfigQuery{
		Scenario:  models.IntegrationScenarioEmbedding,
		Operation: embeddingOperationCreate,
	})
	if err != nil {
		setIndexError := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed,
			fmt.Sprintf("embedding config missing: %v - configure an embedding provider in Parameter > Embedding (scenario=embedding, operation=embedding_create)", err))
		if setIndexError != nil {
			return fmt.Errorf("set document status to failed: %w (original: %v)", setIndexError, err)
		}
		return fmt.Errorf("load embedding config: %w", err)
	}

	// Get embedding adapter
	adapter, ok := registeredEmbeddingAdapter(embedCfg.Channel.AdapterKey)
	if !ok {
		setIndexError := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed,
			fmt.Sprintf("embedding adapter not registered: %s", embedCfg.Channel.AdapterKey))
		if setIndexError != nil {
			return fmt.Errorf("set document status to failed: %w", setIndexError)
		}
		return fmt.Errorf("embedding adapter not registered: %s", embedCfg.Channel.AdapterKey)
	}

	// Build embedding provider config
	providerCfg, err := embeddingProviderConfig(embedCfg)
	if err != nil {
		setIndexError := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed,
			fmt.Sprintf("embedding provider config invalid: %v", err))
		if setIndexError != nil {
			return fmt.Errorf("set document status to failed: %w (original: %v)", setIndexError, err)
		}
		return fmt.Errorf("build embedding provider config: %w", err)
	}

	// Collect chunk texts for batch embedding
	chunkTexts := make([]string, len(chunks))
	for i, c := range chunks {
		chunkTexts[i] = c.Content
	}

	// Generate embeddings
	embedResult, err := adapter.Embed(ctx, providerCfg, embedding.EmbedRequest{
		Operation: embeddingOperationCreate,
		Texts:     chunkTexts,
		Params:    providerCfg.ModelSettings,
	})
	if err != nil {
		setIndexError := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed,
			fmt.Sprintf("embedding generation failed: %v", err))
		if setIndexError != nil {
			return fmt.Errorf("set document status to failed: %w (original: %v)", setIndexError, err)
		}
		return fmt.Errorf("generate embeddings: %w", err)
	}

	if len(embedResult.Vectors) != len(chunks) {
		setIndexError := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed,
			fmt.Sprintf("embedding result count mismatch: got %d vectors for %d chunks", len(embedResult.Vectors), len(chunks)))
		if setIndexError != nil {
			return fmt.Errorf("set document status to failed: %w (original mismatch)", setIndexError)
		}
		return fmt.Errorf("embedding result count mismatch: got %d, want %d", len(embedResult.Vectors), len(chunks))
	}

	// Prepare chunks for storage
	modelInserts := make([]models.KnowledgeChunkInsert, len(chunks))
	embeddingDimension := embedResult.Dimensions
	if embeddingDimension == 0 && len(embedResult.Vectors) > 0 {
		embeddingDimension = len(embedResult.Vectors[0].Values)
	}
	embeddingModelCode := embedResult.ModelCode
	if embeddingModelCode == "" {
		embeddingModelCode = embedCfg.Model.ModelCode
	}
	embeddingProviderModelID := embedResult.ProviderModelID
	if embeddingProviderModelID == "" {
		embeddingProviderModelID = embedCfg.Model.ProviderModelID
	}

	for i, c := range chunks {
		embeddingJSON, err := json.Marshal(embedResult.Vectors[i].Values)
		if err != nil {
			return fmt.Errorf("marshal embedding for chunk %d: %w", i, err)
		}

		modelInserts[i] = models.KnowledgeChunkInsert{
			ID:                       c.ID,
			SourceID:                 doc.SourceID,
			DocumentID:               documentID,
			ChunkIndex:               i,
			Content:                  c.Content,
			ContentHash:              c.ContentHash,
			TokenCount:               c.TokenCount,
			CharCount:                c.CharCount,
			EmbeddingModelCode:       embeddingModelCode,
			EmbeddingProviderModelID: embeddingProviderModelID,
			EmbeddingDimensions:      embeddingDimension,
			EmbeddingJSON:            string(embeddingJSON),
		}
	}

	// Store chunks and embeddings in the database
	if err := models.ReplaceKnowledgeDocumentChunks(ctx, documentID, modelInserts); err != nil {
		setIndexError := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusFailed,
			fmt.Sprintf("store chunks failed: %v", err))
		if setIndexError != nil {
			return fmt.Errorf("set document status to failed: %w (original: %v)", setIndexError, err)
		}
		return fmt.Errorf("store chunks: %w", err)
	}

	// Also update the source's index status
	if err := models.UpdateKnowledgeIndexStatus(ctx, doc.SourceID, documentID, models.KBIndexStatusIndexed, ""); err != nil {
		return fmt.Errorf("update source index status: %w", err)
	}

	return nil
}

// Ensure SQLiteVecRetriever implements both interfaces.
var (
	_ kb.Retriever = (*SQLiteVecRetriever)(nil)
	_ kb.Indexer   = (*SQLiteVecRetriever)(nil)
)
