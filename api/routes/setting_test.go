package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/routes"
	"github.com/tfnick/go-svelte-starter/api/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/oss"
)

type routeSiteLogoFakeOSSAdapter struct{}

func (routeSiteLogoFakeOSSAdapter) PutObject(_ context.Context, _ oss.ProviderConfig, req oss.PutObjectRequest) (oss.PutObjectResult, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return oss.PutObjectResult{}, err
	}
	return oss.PutObjectResult{Key: req.Key, Size: int64(len(body))}, nil
}

func (routeSiteLogoFakeOSSAdapter) GetObject(context.Context, oss.ProviderConfig, oss.GetObjectRequest) (oss.GetObjectResult, error) {
	return oss.GetObjectResult{Body: io.NopCloser(strings.NewReader(""))}, nil
}

func (routeSiteLogoFakeOSSAdapter) DeleteObject(context.Context, oss.ProviderConfig, oss.DeleteObjectRequest) (oss.DeleteObjectResult, error) {
	return oss.DeleteObjectResult{}, nil
}

func (routeSiteLogoFakeOSSAdapter) PresignObject(context.Context, oss.ProviderConfig, oss.PresignObjectRequest) (oss.PresignObjectResult, error) {
	return oss.PresignObjectResult{}, nil
}

func TestGetSiteSettingsReturnsDefaultLogoDTO(t *testing.T) {
	setupRouteTestDBs(t)

	router := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/settings/site", nil)
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.GetSiteSettings(c); err != nil {
		t.Fatalf("get site settings: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool                        `json:"success"`
		Data    routes.SiteSettingsResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success || envelope.Data.LogoURL != "/logo.png" || envelope.Data.LogoConfigured {
		t.Fatalf("unexpected settings response: %s", rec.Body.String())
	}
	if envelope.Data.LogoUploadAvailable || envelope.Data.LogoUploadUnavailableReason == "" {
		t.Fatalf("expected unavailable logo upload state: %s", rec.Body.String())
	}
}

func TestUploadSiteLogoReturnsConfiguredLogoDTO(t *testing.T) {
	setupRouteTestDBs(t)
	adapterKey := "oss.test.route.site_logo"
	if err := usecase.RegisterOSSAdapter(adapterKey, routeSiteLogoFakeOSSAdapter{}); err != nil {
		t.Fatalf("register OSS adapter: %v", err)
	}
	seedRoutePrimaryOSSChannel(t, adapterKey)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	router := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/settings/site/logo", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.UploadSiteLogo(c); err != nil {
		t.Fatalf("upload site logo: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool                        `json:"success"`
		Data    routes.SiteSettingsResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success || !envelope.Data.LogoConfigured || !strings.HasPrefix(envelope.Data.LogoURL, "/api/settings/public/logo?v=") {
		t.Fatalf("unexpected upload response: %s", rec.Body.String())
	}
	if !envelope.Data.LogoUploadAvailable || envelope.Data.LogoUploadUnavailableReason != "" {
		t.Fatalf("expected available logo upload state: %s", rec.Body.String())
	}
}

func TestUploadSiteLogoWithoutPrimaryOSSReturnsValidationError(t *testing.T) {
	setupRouteTestDBs(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	router := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/settings/site/logo", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := router.NewContext(req, rec)

	if err := routes.UploadSiteLogo(c); err != nil {
		t.Fatalf("upload site logo: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"primary OSS provider is not configured"`) {
		t.Fatalf("expected primary OSS validation message, got %s", rec.Body.String())
	}
}

func seedRoutePrimaryOSSChannel(t *testing.T, adapterKey string) {
	t.Helper()

	credential, err := models.CreateIntegrationCredential(t.Context(), models.CreateIntegrationCredentialCmd{
		CredentialType: "s3_access_key",
		ValueText:      `{"access_key_id":"ak-route-logo","secret_access_key":"sk-route-logo"}`,
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create OSS credential: %v", err)
	}
	if _, err := models.CreateIntegrationChannel(t.Context(), models.CreateIntegrationChannelCmd{
		Scenario:     models.IntegrationScenarioOSS,
		ChannelCode:  "route-site-logo-primary",
		ProviderCode: "cloudflare_r2",
		AdapterKey:   adapterKey,
		Environment:  "test",
		Enabled:      true,
		Priority:     1,
		CredentialID: credential.ID,
		IsPrimary:    true,
		ConfigJSON:   `{"endpoint_url":"https://r2.example.com","bucket":"assets","region":"auto"}`,
		MetadataJSON: "{}",
	}); err != nil {
		t.Fatalf("create primary OSS channel: %v", err)
	}
}
