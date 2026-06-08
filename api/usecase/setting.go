package usecase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/tfnick/go-svelte-starter/api/framework/data/modelerror"
	"github.com/tfnick/go-svelte-starter/api/framework/integrations/providererror"
	"github.com/tfnick/go-svelte-starter/api/framework/timefmt"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/oss"
)

const (
	defaultSiteLogoURL = "/logo.png"
	siteLogoSettingKey = "site.logo"
	maxSiteLogoBytes   = 2 * 1024 * 1024
)

type SiteSettingsQry struct{}

type SaveSiteLogoCmd struct {
	Filename    string
	ContentType string
	Size        int64
	Body        io.Reader
}

type SiteLogoObjectQry struct{}

type SiteSettingsCo struct {
	LogoURL                     string
	LogoConfigured              bool
	LogoUpdatedAt               string
	LogoUploadAvailable         bool
	LogoUploadUnavailableReason string
}

type SiteLogoObjectCo struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

type siteLogoMetadata struct {
	ObjectKey    string `json:"object_key"`
	ContentType  string `json:"content_type"`
	Size         int64  `json:"size"`
	UpdatedAt    string `json:"updated_at"`
	ChannelCode  string `json:"channel_code"`
	ProviderCode string `json:"provider_code"`
	AdapterKey   string `json:"adapter_key"`
}

func GetSiteSettings(ctx fwusecase.Context, _ SiteSettingsQry) (SiteSettingsCo, error) {
	meta, found, err := loadSiteLogoMetadata(ctx)
	if err != nil {
		return SiteSettingsCo{}, err
	}
	uploadState := resolveSiteLogoUploadState(ctx)
	if !found {
		return withSiteLogoUploadState(defaultSiteSettings(), uploadState), nil
	}
	return withSiteLogoUploadState(siteSettingsFromLogoMetadata(meta), uploadState), nil
}

func SaveSiteLogo(ctx fwusecase.Context, cmd SaveSiteLogoCmd) (SiteSettingsCo, error) {
	if cmd.Body == nil {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeValidation, "logo file is required", nil)
	}
	if cmd.Size > maxSiteLogoBytes {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeValidation, "logo file is too large", nil)
	}

	payload, err := io.ReadAll(io.LimitReader(cmd.Body, maxSiteLogoBytes+1))
	if err != nil {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to read logo file", err)
	}
	if len(payload) == 0 {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeValidation, "logo file is required", nil)
	}
	if len(payload) > maxSiteLogoBytes {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeValidation, "logo file is too large", nil)
	}

	contentType, err := normalizeSiteLogoContentType(cmd.ContentType, payload)
	if err != nil {
		return SiteSettingsCo{}, err
	}

	provider, err := primarySiteLogoProvider(ctx)
	if err != nil {
		return SiteSettingsCo{}, err
	}
	adapter, ok := registeredOSSAdapter(provider.Config.AdapterKey)
	if !ok {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeInternal, "logo storage is not configured", fmt.Errorf("OSS adapter not registered: %s", provider.Config.AdapterKey))
	}

	objectKey := "settings/site-logo" + siteLogoExtension(contentType)
	result, err := adapter.PutObject(ctx.Std(), provider.Config, oss.PutObjectRequest{
		Key:         objectKey,
		Body:        bytes.NewReader(payload),
		Size:        int64(len(payload)),
		ContentType: contentType,
		Metadata: map[string]string{
			"filename": strings.TrimSpace(cmd.Filename),
		},
	})
	if err != nil {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to store logo", err)
	}
	if strings.TrimSpace(result.Key) != "" {
		objectKey = result.Key
	}

	meta := siteLogoMetadata{
		ObjectKey:    objectKey,
		ContentType:  contentType,
		Size:         int64(len(payload)),
		UpdatedAt:    timefmt.RFC3339Nano(timefmt.NowUTC()),
		ChannelCode:  provider.Config.ChannelCode,
		ProviderCode: provider.Config.ProviderCode,
		AdapterKey:   provider.Config.AdapterKey,
	}
	encoded, err := json.Marshal(meta)
	if err != nil {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to encode logo settings", err)
	}
	if _, err := models.UpsertAppSetting(ctx.Std(), siteLogoSettingKey, string(encoded)); err != nil {
		return SiteSettingsCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to save logo settings", err)
	}

	return withSiteLogoUploadState(siteSettingsFromLogoMetadata(meta), siteLogoUploadAvailability{Available: true}), nil
}

func GetSiteLogoObject(ctx fwusecase.Context, _ SiteLogoObjectQry) (SiteLogoObjectCo, error) {
	meta, found, err := loadSiteLogoMetadata(ctx)
	if err != nil {
		return SiteLogoObjectCo{}, err
	}
	if !found {
		return SiteLogoObjectCo{}, fwusecase.E(fwusecase.CodeNotFound, "site logo is not configured", nil)
	}

	provider, err := siteLogoProviderFromMetadata(ctx, meta)
	if err != nil {
		return SiteLogoObjectCo{}, err
	}
	adapter, ok := registeredOSSAdapter(provider.Config.AdapterKey)
	if !ok {
		return SiteLogoObjectCo{}, fwusecase.E(fwusecase.CodeInternal, "logo storage is not configured", fmt.Errorf("OSS adapter not registered: %s", provider.Config.AdapterKey))
	}

	result, err := adapter.GetObject(ctx.Std(), provider.Config, oss.GetObjectRequest{Key: meta.ObjectKey})
	if err != nil {
		if providerErr, ok := providererror.From(err); ok && providerErr.Category == providererror.CategoryPermanent {
			return SiteLogoObjectCo{}, fwusecase.E(fwusecase.CodeNotFound, "site logo is not configured", err)
		}
		return SiteLogoObjectCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load logo", err)
	}
	if result.Body == nil {
		return SiteLogoObjectCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load logo", fmt.Errorf("OSS object body is empty"))
	}

	contentType := strings.TrimSpace(result.ContentType)
	if contentType == "" {
		contentType = meta.ContentType
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return SiteLogoObjectCo{
		Body:        result.Body,
		ContentType: contentType,
		Size:        result.Size,
	}, nil
}

func loadSiteLogoMetadata(ctx fwusecase.Context) (siteLogoMetadata, bool, error) {
	setting, err := models.GetAppSetting(ctx.Std(), siteLogoSettingKey)
	if err != nil {
		if errors.Is(err, modelerror.ErrNotFound) {
			return siteLogoMetadata{}, false, nil
		}
		return siteLogoMetadata{}, false, fwusecase.E(fwusecase.CodeInternal, "failed to load site settings", err)
	}

	var meta siteLogoMetadata
	if err := json.Unmarshal([]byte(setting.ValueJSON), &meta); err != nil {
		return siteLogoMetadata{}, false, fwusecase.E(fwusecase.CodeInternal, "failed to parse site settings", err)
	}
	if strings.TrimSpace(meta.ObjectKey) == "" {
		return siteLogoMetadata{}, false, nil
	}
	return meta, true, nil
}

func defaultSiteSettings() SiteSettingsCo {
	return SiteSettingsCo{
		LogoURL:        defaultSiteLogoURL,
		LogoConfigured: false,
	}
}

func siteSettingsFromLogoMetadata(meta siteLogoMetadata) SiteSettingsCo {
	logoURL := "/api/settings/public/logo"
	if meta.UpdatedAt != "" {
		logoURL += "?v=" + url.QueryEscape(meta.UpdatedAt)
	}
	return SiteSettingsCo{
		LogoURL:        logoURL,
		LogoConfigured: true,
		LogoUpdatedAt:  meta.UpdatedAt,
	}
}

type siteLogoUploadAvailability struct {
	Available bool
	Reason    string
}

type siteLogoProvider struct {
	Config oss.ProviderConfig
}

type ossChannelConfigJSON struct {
	EndpointURL   string `json:"endpoint_url"`
	Bucket        string `json:"bucket"`
	Region        string `json:"region"`
	PublicBaseURL string `json:"public_base_url"`
	KeyPrefix     string `json:"key_prefix"`
}

type ossChannelCredentialJSON struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

func withSiteLogoUploadState(settings SiteSettingsCo, state siteLogoUploadAvailability) SiteSettingsCo {
	settings.LogoUploadAvailable = state.Available
	settings.LogoUploadUnavailableReason = state.Reason
	return settings
}

func resolveSiteLogoUploadState(ctx fwusecase.Context) siteLogoUploadAvailability {
	provider, err := primarySiteLogoProvider(ctx)
	if err != nil {
		return siteLogoUploadAvailability{Available: false, Reason: "Primary OSS provider is not configured"}
	}
	if _, ok := registeredOSSAdapter(provider.Config.AdapterKey); !ok {
		return siteLogoUploadAvailability{Available: false, Reason: "Primary OSS provider adapter is not available"}
	}
	return siteLogoUploadAvailability{Available: true}
}

func primarySiteLogoProvider(ctx fwusecase.Context) (siteLogoProvider, error) {
	channel, err := models.GetEnabledPrimaryOSSChannelConfig(ctx.Std())
	if err != nil {
		if errors.Is(err, modelerror.ErrNotFound) {
			return siteLogoProvider{}, fwusecase.E(fwusecase.CodeValidation, "primary OSS provider is not configured", err)
		}
		return siteLogoProvider{}, fwusecase.E(fwusecase.CodeInternal, "failed to load primary OSS provider", err)
	}
	return siteLogoProviderFromChannel(channel)
}

func siteLogoProviderFromMetadata(ctx fwusecase.Context, meta siteLogoMetadata) (siteLogoProvider, error) {
	if strings.TrimSpace(meta.AdapterKey) == "" || strings.TrimSpace(meta.ChannelCode) == "" {
		return primarySiteLogoProvider(ctx)
	}

	channel, err := models.GetOSSChannelConfigByCodeAndAdapter(ctx.Std(), strings.TrimSpace(meta.ChannelCode), strings.TrimSpace(meta.AdapterKey))
	if err != nil {
		if errors.Is(err, modelerror.ErrNotFound) {
			return siteLogoProvider{}, fwusecase.E(fwusecase.CodeInternal, "logo storage is not configured", err)
		}
		return siteLogoProvider{}, fwusecase.E(fwusecase.CodeInternal, "failed to load OSS provider", err)
	}
	return siteLogoProviderFromChannel(channel)
}

func siteLogoProviderFromChannel(channel models.IntegrationChannelConfig) (siteLogoProvider, error) {
	var cfg ossChannelConfigJSON
	if err := json.Unmarshal([]byte(strings.TrimSpace(channel.ConfigJSON)), &cfg); err != nil {
		return siteLogoProvider{}, fwusecase.E(fwusecase.CodeInternal, "OSS provider config is invalid", err)
	}
	var credential ossChannelCredentialJSON
	if err := json.Unmarshal([]byte(strings.TrimSpace(channel.CredentialValue)), &credential); err != nil {
		return siteLogoProvider{}, fwusecase.E(fwusecase.CodeInternal, "OSS provider credential is invalid", err)
	}

	providerCfg := oss.ProviderConfig{
		ChannelCode:     channel.ChannelCode,
		ProviderCode:    channel.ProviderCode,
		AdapterKey:      channel.AdapterKey,
		EndpointURL:     strings.TrimSpace(cfg.EndpointURL),
		Bucket:          strings.TrimSpace(cfg.Bucket),
		Region:          strings.TrimSpace(cfg.Region),
		PublicBaseURL:   strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"),
		KeyPrefix:       strings.Trim(strings.TrimSpace(cfg.KeyPrefix), "/"),
		AccessKeyID:     strings.TrimSpace(credential.AccessKeyID),
		SecretAccessKey: strings.TrimSpace(credential.SecretAccessKey),
	}
	if providerCfg.EndpointURL == "" || providerCfg.Bucket == "" {
		return siteLogoProvider{}, fwusecase.E(fwusecase.CodeInternal, "OSS provider config is invalid", fmt.Errorf("endpoint_url and bucket are required"))
	}
	if providerCfg.AccessKeyID == "" || providerCfg.SecretAccessKey == "" {
		return siteLogoProvider{}, fwusecase.E(fwusecase.CodeInternal, "OSS provider credential is invalid", fmt.Errorf("access key id and secret access key are required"))
	}
	return siteLogoProvider{Config: providerCfg}, nil
}

func normalizeSiteLogoContentType(_ string, payload []byte) (string, error) {
	if len(payload) >= 8 && bytes.Equal(payload[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png", nil
	}
	if len(payload) >= 3 && payload[0] == 0xff && payload[1] == 0xd8 && payload[2] == 0xff {
		return "image/jpeg", nil
	}
	if len(payload) >= 12 && string(payload[:4]) == "RIFF" && string(payload[8:12]) == "WEBP" {
		return "image/webp", nil
	}

	return "", fwusecase.E(fwusecase.CodeValidation, "logo image type is not supported", nil)
}

func siteLogoExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}
