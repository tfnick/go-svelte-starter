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

	router.GET("/*", func(c echo.Context) error {
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
	})
}

func redirectToFrontendDevServer(c echo.Context, frontendDevURL string) error {
	target := strings.TrimRight(frontendDevURL, "/") + c.Request().URL.RequestURI()
	return c.Redirect(http.StatusTemporaryRedirect, target)
}

func serveEmbeddedFrontend(c echo.Context, dist fs.FS) error {
	requestPath := strings.TrimPrefix(c.Request().URL.Path, "/")
	if requestPath == "" {
		requestPath = "index.html"
	}

	cleanPath := path.Clean(requestPath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "../") {
		return c.NoContent(http.StatusNotFound)
	}

	if file, err := dist.Open(cleanPath); err == nil {
		defer file.Close()
		if info, statErr := file.Stat(); statErr == nil && !info.IsDir() {
			return c.Stream(http.StatusOK, contentType(cleanPath), file)
		}
	}

	index, err := dist.Open("index.html")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "embedded frontend index.html is missing",
		})
	}
	defer index.Close()

	return c.Stream(http.StatusOK, "text/html; charset=utf-8", index)
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
