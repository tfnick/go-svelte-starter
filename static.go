package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed frontend/dist
var frontendDistFS embed.FS

func registerFrontendRoutes(router *echo.Echo, isDevelopment bool, frontendDevURL string) {
	dist, err := fs.Sub(frontendDistFS, "frontend/dist")
	if err != nil {
		router.Logger.Fatal(err)
	}

	router.RouteNotFound("/api/*", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "API endpoint not found",
		})
	})

	frontendHandler := func(c echo.Context) error {
		requestPath := c.Request().URL.Path
		if strings.HasPrefix(requestPath, "/api/") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "API endpoint not found",
			})
		}

		if isDevelopment {
			return redirectToFrontendDevServer(c, frontendDevURL)
		}

		return serveEmbeddedFrontend(c, dist)
	}

	router.GET("/app", frontendHandler)
	router.GET("/app/*", frontendHandler)
	router.GET("/assets/*", func(c echo.Context) error {
		return serveEmbeddedFrontendAsset(c, dist)
	})
	router.GET("/logo.png", func(c echo.Context) error {
		return serveEmbeddedFrontendAsset(c, dist)
	})

	for legacyPath, appPath := range legacyFrontendRedirects() {
		router.GET(legacyPath, redirectLegacyFrontendRoute(appPath))
	}
}

func redirectToFrontendDevServer(c echo.Context, frontendDevURL string) error {
	target := strings.TrimRight(frontendDevURL, "/") + c.Request().URL.RequestURI()
	return c.Redirect(http.StatusTemporaryRedirect, target)
}

func serveEmbeddedFrontend(c echo.Context, dist fs.FS) error {
	index, err := dist.Open("index.html")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "embedded frontend index.html is missing",
		})
	}
	defer index.Close()

	return c.Stream(http.StatusOK, "text/html; charset=utf-8", index)
}

func serveEmbeddedFrontendAsset(c echo.Context, dist fs.FS) error {
	requestPath := strings.TrimPrefix(c.Request().URL.Path, "/")
	cleanPath := path.Clean(requestPath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "../") {
		return c.NoContent(http.StatusNotFound)
	}

	file, err := dist.Open(cleanPath)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	defer file.Close()

	if info, statErr := file.Stat(); statErr != nil || info.IsDir() {
		return c.NoContent(http.StatusNotFound)
	}

	return c.Stream(http.StatusOK, contentType(cleanPath), file)
}

func legacyFrontendRedirects() map[string]string {
	return map[string]string{
		"/index.html":                "/",
		"/dashboard":                 "/app",
		"/login":                     "/app/login",
		"/login.html":                "/app/login",
		"/login/oauth/callback":      "/app/login/oauth/callback",
		"/login-oauth-callback.html": "/app/login/oauth/callback",
		"/register":                  "/app/register",
		"/register.html":             "/app/register",
		"/forgot-password":           "/app/forgot-password",
		"/forgot-password.html":      "/app/forgot-password",
		"/reset-password":            "/app/reset-password",
		"/reset-password.html":       "/app/reset-password",
		"/orders":                    "/app/orders",
		"/orders.html":               "/app/orders",
		"/products":                  "/app/products",
		"/products.html":             "/app/products",
		"/users":                     "/app/users",
		"/users.html":                "/app/users",
		"/scheduler":                 "/app/scheduler",
		"/scheduler.html":            "/app/scheduler",
		"/events":                    "/app/events",
		"/events.html":               "/app/events",
		"/experiments":               "/app/experiments",
		"/experiments.html":          "/app/experiments",
		"/dictionary":                "/app/dictionary",
		"/dictionary.html":           "/app/dictionary",
		"/parameters":                "/app/parameters",
		"/parameters.html":           "/app/parameters",
		"/notifications":             "/app/notifications",
		"/notifications.html":        "/app/notifications",
		"/settings":                  "/app/settings",
		"/settings.html":             "/app/settings",
		"/variables":                 "/app/variables",
		"/variables.html":            "/app/variables",
	}
}

func redirectLegacyFrontendRoute(targetPath string) echo.HandlerFunc {
	return func(c echo.Context) error {
		target := targetPath
		if query := c.Request().URL.RawQuery; query != "" {
			target += "?" + query
		}
		return c.Redirect(http.StatusTemporaryRedirect, target)
	}
}

func contentType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".json":
		return "application/json"
	case ".html":
		return "text/html; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
