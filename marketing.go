package main

import (
	"embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

const (
	marketingSiteName = "Svelte Go Starter"
	marketingCSSPath  = "/marketing/assets/marketing.css"
)

//go:embed marketing/templates/*.html marketing/assets/*
var marketingFS embed.FS

type marketingRenderer struct {
	templatesOnce   sync.Once
	parsedTemplates *template.Template
	templatesErr    error
	assets          fs.FS
}

type marketingPage struct {
	Key         string
	Path        string
	Title       string
	Description string
}

type marketingProduct struct {
	ID           string
	Name         string
	Description  string
	PriceLabel   string
	BillingLabel string
	CheckoutURL  string
}

type marketingPageData struct {
	SiteName     string
	LogoURL      string
	BaseURL      string
	CSSPath      string
	Page         marketingPage
	CanonicalURL string
	Products     []marketingProduct
	JSONLD       template.JS
	Year         int
}

func registerMarketingRoutes(router *echo.Echo) {
	assets, err := fs.Sub(marketingFS, "marketing/assets")
	if err != nil {
		router.Logger.Fatal(err)
	}

	renderer := &marketingRenderer{assets: assets}
	router.GET("/", renderer.renderPage(marketingPage{
		Key:         "home",
		Path:        "/",
		Title:       "Svelte Go Starter | One Binary SaaS Starter",
		Description: "Launch a Go and Svelte SaaS with server-rendered marketing pages, checkout, auth, and an embedded dashboard.",
	}))
	router.GET("/pricing", renderer.renderPage(marketingPage{
		Key:         "pricing",
		Path:        "/pricing",
		Title:       "Pricing | Svelte Go Starter",
		Description: "Compare checkout-ready SaaS starter plans backed by the same product catalog used by the application.",
	}))
	router.GET("/features", renderer.renderPage(marketingPage{
		Key:         "features",
		Path:        "/features",
		Title:       "Features | Svelte Go Starter",
		Description: "Explore the production-oriented Go and Svelte starter features for auth, payment, settings, events, and operations.",
	}))
	router.GET("/robots.txt", renderer.robots)
	router.GET("/sitemap.xml", renderer.sitemap)
	router.GET("/marketing/assets/*", renderer.asset)
}

func (r *marketingRenderer) renderPage(page marketingPage) echo.HandlerFunc {
	return func(c echo.Context) error {
		tmpl, err := r.loadTemplates()
		if err != nil {
			return c.String(http.StatusInternalServerError, "marketing templates are unavailable")
		}

		data := r.pageData(c, page)
		c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
		c.Response().WriteHeader(http.StatusOK)
		return tmpl.ExecuteTemplate(c.Response(), page.Key, data)
	}
}

func (r *marketingRenderer) loadTemplates() (*template.Template, error) {
	r.templatesOnce.Do(func() {
		funcs := template.FuncMap{
			"lower": strings.ToLower,
		}
		r.parsedTemplates, r.templatesErr = template.New("marketing").Funcs(funcs).ParseFS(marketingFS, "marketing/templates/*.html")
	})
	return r.parsedTemplates, r.templatesErr
}

func (r *marketingRenderer) pageData(c echo.Context, page marketingPage) marketingPageData {
	baseURL := marketingBaseURL(c)
	logoURL := marketingLogoURL(c)
	products := marketingProducts(c)
	data := marketingPageData{
		SiteName:     marketingSiteName,
		LogoURL:      logoURL,
		BaseURL:      baseURL,
		CSSPath:      marketingCSSPath,
		Page:         page,
		CanonicalURL: baseURL + page.Path,
		Products:     products,
		Year:         time.Now().Year(),
	}
	data.JSONLD = marketingJSONLD(data)
	return data
}

func marketingBaseURL(c echo.Context) string {
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_PUBLIC_BASE_URL")), "/"); configured != "" {
		return configured
	}

	req := c.Request()
	scheme := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = "http"
		if req.TLS != nil {
			scheme = "https"
		}
	}
	host := strings.TrimSpace(req.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = req.Host
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func marketingLogoURL(c echo.Context) string {
	ctx := fwusecase.NewContext(c.Request().Context(), fwusecase.SurfaceInternalAPI)
	settings, err := usecase.GetSiteSettings(ctx, usecase.SiteSettingsQry{})
	if err != nil || strings.TrimSpace(settings.LogoURL) == "" {
		return "/logo.png"
	}
	return settings.LogoURL
}

func marketingProducts(c echo.Context) []marketingProduct {
	ctx := fwusecase.NewContext(c.Request().Context(), fwusecase.SurfaceInternalAPI)
	products, err := usecase.ListProducts(ctx)
	if err != nil {
		c.Logger().Warnf("failed to load marketing products: %v", err)
		return nil
	}

	result := make([]marketingProduct, 0, len(products))
	for _, product := range products {
		if !product.Enabled || strings.TrimSpace(product.CreemProductID) == "" {
			continue
		}
		result = append(result, marketingProduct{
			ID:           product.ID,
			Name:         product.Name,
			Description:  product.Description,
			PriceLabel:   marketingPrice(product.Price, product.Currency),
			BillingLabel: marketingBilling(product.BillingType, product.SubscriptionInterval),
			CheckoutURL:  "/app/checkout?product_id=" + url.QueryEscape(product.ID),
		})
	}
	return result
}

func marketingPrice(cents int64, currency string) string {
	if cents <= 0 {
		return "Contact us"
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = "USD"
	}
	switch currency {
	case "USD":
		return fmt.Sprintf("$%.2f", float64(cents)/100)
	case "CNY":
		return fmt.Sprintf("CNY %.2f", float64(cents)/100)
	default:
		return fmt.Sprintf("%s %.2f", currency, float64(cents)/100)
	}
}

func marketingBilling(billingType string, interval string) string {
	switch strings.TrimSpace(billingType) {
	case usecase.ProductBillingTypeSubscription:
		switch strings.TrimSpace(interval) {
		case usecase.SubscriptionIntervalYear:
			return "per year"
		case usecase.SubscriptionIntervalThreeMonths:
			return "per 3 months"
		case usecase.SubscriptionIntervalSixMonths:
			return "per 6 months"
		default:
			return "per month"
		}
	default:
		return "one-time"
	}
}

func marketingJSONLD(data marketingPageData) template.JS {
	offers := make([]map[string]string, 0, len(data.Products))
	for _, product := range data.Products {
		offers = append(offers, map[string]string{
			"@type": "Offer",
			"name":  product.Name,
			"url":   data.BaseURL + product.CheckoutURL,
		})
	}
	payload := map[string]any{
		"@context":            "https://schema.org",
		"@type":               "SoftwareApplication",
		"name":                data.SiteName,
		"applicationCategory": "BusinessApplication",
		"operatingSystem":     "Web",
		"url":                 data.CanonicalURL,
		"description":         data.Page.Description,
	}
	if len(offers) > 0 {
		payload["offers"] = offers
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(encoded)
}

func (r *marketingRenderer) robots(c echo.Context) error {
	baseURL := marketingBaseURL(c)
	body := "User-agent: *\nAllow: /\n"
	if baseURL != "" {
		body += "Sitemap: " + baseURL + "/sitemap.xml\n"
	}
	return c.Blob(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

func (r *marketingRenderer) sitemap(c echo.Context) error {
	baseURL := marketingBaseURL(c)
	urls := []sitemapURL{
		{Loc: baseURL + "/", ChangeFreq: "weekly", Priority: "1.0"},
		{Loc: baseURL + "/pricing", ChangeFreq: "weekly", Priority: "0.8"},
		{Loc: baseURL + "/features", ChangeFreq: "monthly", Priority: "0.7"},
	}
	payload, err := xml.MarshalIndent(sitemapURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}, "", "  ")
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to build sitemap")
	}
	payload = append([]byte(xml.Header), payload...)
	return c.Blob(http.StatusOK, "application/xml; charset=utf-8", payload)
}

func (r *marketingRenderer) asset(c echo.Context) error {
	requestPath := strings.TrimPrefix(c.Param("*"), "/")
	cleanPath := path.Clean(requestPath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "../") {
		return c.NoContent(http.StatusNotFound)
	}

	file, err := r.assets.Open(cleanPath)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	defer file.Close()

	if info, statErr := file.Stat(); statErr != nil || info.IsDir() {
		return c.NoContent(http.StatusNotFound)
	}
	return c.Stream(http.StatusOK, contentType(cleanPath), file)
}
