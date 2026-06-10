package routes

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	httpresponse "github.com/tfnick/go-svelte-starter/api/framework/http/response"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

type SiteSettingsResponse struct {
	LogoURL                     string `json:"logo_url"`
	LogoConfigured              bool   `json:"logo_configured"`
	LogoUpdatedAt               string `json:"logo_updated_at"`
	LogoUploadAvailable         bool   `json:"logo_upload_available"`
	LogoUploadUnavailableReason string `json:"logo_upload_unavailable_reason"`
}

func GetSiteSettings(c echo.Context) error {
	settings, err := usecase.GetSiteSettings(fwcontext.InternalUsecaseContext(c), usecase.SiteSettingsQry{})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}
	return httpresponse.OK(c, toSiteSettingsResponse(settings))
}

func UploadSiteLogo(c echo.Context) error {
	fileHeader, err := c.FormFile("logo")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return httpresponse.InternalUsecaseError(c, fwusecase.E(fwusecase.CodeValidation, "logo file is required", err))
		}
		return httpresponse.InternalUsecaseError(c, fwusecase.E(fwusecase.CodeValidation, "logo file is required", err))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return httpresponse.InternalUsecaseError(c, fwusecase.E(fwusecase.CodeInternal, "failed to read logo file", err))
	}
	defer file.Close()

	settings, err := usecase.SaveSiteLogo(fwcontext.InternalUsecaseContext(c), usecase.SaveSiteLogoCmd{
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get(echo.HeaderContentType),
		Size:        fileHeader.Size,
		Body:        file,
	})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}
	return httpresponse.OK(c, toSiteSettingsResponse(settings))
}

func GetPublicSiteLogo(c echo.Context) error {
	logo, err := usecase.GetSiteLogoObject(fwcontext.InternalUsecaseContext(c), usecase.SiteLogoObjectQry{})
	if err != nil {
		if fwusecase.CodeOf(err) == fwusecase.CodeNotFound {
			return c.Redirect(http.StatusFound, "/logo.png")
		}
		return httpresponse.InternalUsecaseError(c, err)
	}
	defer logo.Body.Close()

	c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=300")
	return c.Stream(http.StatusOK, logo.ContentType, logo.Body)
}

type WorkerLimitResponse struct {
	Limit int `json:"limit"`
}

func GetWorkerLimit(c echo.Context) error {
	limit, err := usecase.GetWorkerLimit(fwcontext.InternalUsecaseContext(c))
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}
	return httpresponse.OK(c, WorkerLimitResponse{Limit: limit})
}

type SaveWorkerLimitRequest struct {
	Limit int `json:"limit"`
}

func SaveWorkerLimit(c echo.Context) error {
	var req SaveWorkerLimitRequest
	if err := c.Bind(&req); err != nil {
		return httpresponse.BadRequest(c, "invalid request data")
	}

	limit, err := usecase.SaveWorkerLimit(fwcontext.InternalUsecaseContext(c), req.Limit)
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}
	return httpresponse.OK(c, WorkerLimitResponse{Limit: limit})
}

func toSiteSettingsResponse(settings usecase.SiteSettingsCo) SiteSettingsResponse {
	return SiteSettingsResponse{
		LogoURL:                     settings.LogoURL,
		LogoConfigured:              settings.LogoConfigured,
		LogoUpdatedAt:               settings.LogoUpdatedAt,
		LogoUploadAvailable:         settings.LogoUploadAvailable,
		LogoUploadUnavailableReason: settings.LogoUploadUnavailableReason,
	}
}
