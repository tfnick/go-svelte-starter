package usecase_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/oss"
)

type siteLogoFakeOSSAdapter struct {
	putKey         string
	putConfig      oss.ProviderConfig
	putContentType string
	getConfig      oss.ProviderConfig
	objects        map[string][]byte
}

func (a *siteLogoFakeOSSAdapter) PutObject(_ context.Context, cfg oss.ProviderConfig, req oss.PutObjectRequest) (oss.PutObjectResult, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return oss.PutObjectResult{}, err
	}
	if a.objects == nil {
		a.objects = map[string][]byte{}
	}
	a.putKey = req.Key
	a.putConfig = cfg
	a.putContentType = req.ContentType
	a.objects[req.Key] = body
	return oss.PutObjectResult{Key: req.Key, Size: int64(len(body))}, nil
}

func (a *siteLogoFakeOSSAdapter) GetObject(_ context.Context, cfg oss.ProviderConfig, req oss.GetObjectRequest) (oss.GetObjectResult, error) {
	a.getConfig = cfg
	body := a.objects[req.Key]
	return oss.GetObjectResult{
		Key:         req.Key,
		Body:        io.NopCloser(bytes.NewReader(body)),
		ContentType: a.putContentType,
		Size:        int64(len(body)),
	}, nil
}

func (a *siteLogoFakeOSSAdapter) DeleteObject(context.Context, oss.ProviderConfig, oss.DeleteObjectRequest) (oss.DeleteObjectResult, error) {
	return oss.DeleteObjectResult{}, nil
}

func (a *siteLogoFakeOSSAdapter) PresignObject(context.Context, oss.ProviderConfig, oss.PresignObjectRequest) (oss.PresignObjectResult, error) {
	return oss.PresignObjectResult{}, nil
}

func TestGetSiteSettingsDefaultsToPublicLogo(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	settings, err := usecase.GetSiteSettings(ctx, usecase.SiteSettingsQry{})
	if err != nil {
		t.Fatalf("get site settings: %v", err)
	}
	if settings.LogoURL != "/logo.png" || settings.LogoConfigured {
		t.Fatalf("expected default logo settings, got %#v", settings)
	}
	if settings.LogoUploadAvailable || settings.LogoUploadUnavailableReason == "" {
		t.Fatalf("expected logo upload to be unavailable without primary OSS, got %#v", settings)
	}
}

func TestSaveSiteLogoRequiresPrimaryOSSProvider(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	logo := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	_, err := usecase.SaveSiteLogo(ctx, usecase.SaveSiteLogoCmd{
		Filename:    "logo.png",
		ContentType: "image/png",
		Size:        int64(len(logo)),
		Body:        bytes.NewReader(logo),
	})
	if err == nil {
		t.Fatalf("expected missing primary OSS provider error")
	}
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation error code, got %q: %v", fwusecase.CodeOf(err), err)
	}
}

func TestSaveSiteLogoUsesOSSPortAndPersistsMetadata(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	adapter := &siteLogoFakeOSSAdapter{}
	adapterKey := "oss.test.site_logo"
	if err := usecase.RegisterOSSAdapter(adapterKey, adapter); err != nil {
		t.Fatalf("register OSS adapter: %v", err)
	}
	seedPrimaryOSSChannel(t, adapterKey)

	logo := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	settings, err := usecase.SaveSiteLogo(ctx, usecase.SaveSiteLogoCmd{
		Filename:    "logo.png",
		ContentType: "image/png",
		Size:        int64(len(logo)),
		Body:        bytes.NewReader(logo),
	})
	if err != nil {
		t.Fatalf("save site logo: %v", err)
	}
	if adapter.putKey != "settings/site-logo.png" {
		t.Fatalf("expected logo object key, got %q", adapter.putKey)
	}
	if adapter.putContentType != "image/png" {
		t.Fatalf("expected image/png content type, got %q", adapter.putContentType)
	}
	if adapter.putConfig.ChannelCode != "site-logo-primary" || adapter.putConfig.EndpointURL != "https://r2.example.com" || adapter.putConfig.Bucket != "assets" {
		t.Fatalf("expected primary OSS provider config, got %#v", adapter.putConfig)
	}
	if adapter.putConfig.AccessKeyID != "ak-site-logo" || adapter.putConfig.SecretAccessKey != "sk-site-logo" {
		t.Fatalf("expected primary OSS credential, got %#v", adapter.putConfig)
	}
	if !settings.LogoConfigured || !strings.HasPrefix(settings.LogoURL, "/api/settings/public/logo?v=") {
		t.Fatalf("expected configured logo URL, got %#v", settings)
	}
	if !settings.LogoUploadAvailable || settings.LogoUploadUnavailableReason != "" {
		t.Fatalf("expected logo upload to be available, got %#v", settings)
	}

	object, err := usecase.GetSiteLogoObject(ctx, usecase.SiteLogoObjectQry{})
	if err != nil {
		t.Fatalf("get site logo object: %v", err)
	}
	defer object.Body.Close()
	got, err := io.ReadAll(object.Body)
	if err != nil {
		t.Fatalf("read logo object: %v", err)
	}
	if !bytes.Equal(got, logo) {
		t.Fatalf("expected stored logo bytes, got %v", got)
	}
	if adapter.getConfig.ChannelCode != adapter.putConfig.ChannelCode || adapter.getConfig.AdapterKey != adapter.putConfig.AdapterKey {
		t.Fatalf("expected readback through persisted OSS provider metadata, got put=%#v get=%#v", adapter.putConfig, adapter.getConfig)
	}
}

func seedPrimaryOSSChannel(t *testing.T, adapterKey string) {
	t.Helper()

	credential, err := models.CreateIntegrationCredential(t.Context(), models.CreateIntegrationCredentialCmd{
		CredentialType: "s3_access_key",
		ValueText:      `{"access_key_id":"ak-site-logo","secret_access_key":"sk-site-logo"}`,
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create OSS credential: %v", err)
	}
	if _, err := models.CreateIntegrationChannel(t.Context(), models.CreateIntegrationChannelCmd{
		Scenario:     models.IntegrationScenarioOSS,
		ChannelCode:  "site-logo-primary",
		ProviderCode: "cloudflare_r2",
		AdapterKey:   adapterKey,
		Environment:  "test",
		Enabled:      true,
		Priority:     1,
		CredentialID: credential.ID,
		IsPrimary:    true,
		ConfigJSON:   `{"endpoint_url":"https://r2.example.com","bucket":"assets","region":"auto","key_prefix":"public"}`,
		MetadataJSON: "{}",
	}); err != nil {
		t.Fatalf("create primary OSS channel: %v", err)
	}
}
