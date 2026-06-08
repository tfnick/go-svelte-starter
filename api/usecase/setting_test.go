package usecase_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/oss"
)

type siteLogoFakeOSSAdapter struct {
	putKey         string
	putContentType string
	objects        map[string][]byte
}

func (a *siteLogoFakeOSSAdapter) PutObject(_ context.Context, _ oss.ProviderConfig, req oss.PutObjectRequest) (oss.PutObjectResult, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return oss.PutObjectResult{}, err
	}
	if a.objects == nil {
		a.objects = map[string][]byte{}
	}
	a.putKey = req.Key
	a.putContentType = req.ContentType
	a.objects[req.Key] = body
	return oss.PutObjectResult{Key: req.Key, Size: int64(len(body))}, nil
}

func (a *siteLogoFakeOSSAdapter) GetObject(_ context.Context, _ oss.ProviderConfig, req oss.GetObjectRequest) (oss.GetObjectResult, error) {
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
}

func TestSaveSiteLogoUsesOSSPortAndPersistsMetadata(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	adapter := &siteLogoFakeOSSAdapter{}
	if err := usecase.RegisterOSSAdapter(usecase.SiteLogoOSSAdapterKey, adapter); err != nil {
		t.Fatalf("register OSS adapter: %v", err)
	}

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
	if !settings.LogoConfigured || !strings.HasPrefix(settings.LogoURL, "/api/settings/public/logo?v=") {
		t.Fatalf("expected configured logo URL, got %#v", settings)
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
}
