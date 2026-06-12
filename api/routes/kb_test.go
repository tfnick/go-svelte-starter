package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/routes"
)

func TestListKBSourcesReturnsDescriptionDTO(t *testing.T) {
	setupRouteTestDBs(t)
	if _, err := models.CreateKnowledgeSource(t.Context(), models.SaveKnowledgeSourceCmd{
		ID:         "route-kb-source",
		Title:      "Route KB Source",
		SourceType: models.KBSourceTypeManual,
		Enabled:    true,
		Content:    "Description used by the edit form.",
	}); err != nil {
		t.Fatalf("create knowledge source: %v", err)
	}

	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/kb/sources", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.ListKBSources(c); err != nil {
		t.Fatalf("list kb sources: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool                      `json:"success"`
		Data    []routes.KBSourceResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success || len(envelope.Data) != 1 {
		t.Fatalf("unexpected envelope: %s", rec.Body.String())
	}
	if envelope.Data[0].Description != "Description used by the edit form." {
		t.Fatalf("expected description in source DTO, got %#v", envelope.Data[0])
	}
}
