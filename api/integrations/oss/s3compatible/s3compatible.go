package s3compatible

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tfnick/go-svelte-starter/api/framework/integrations/providererror"
	"github.com/tfnick/go-svelte-starter/api/usecase/integrations/oss"
)

const (
	defaultRegion = "auto"
	serviceName   = "s3"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Adapter struct {
	client HTTPDoer
	now    func() time.Time
}

func NewAdapter(client HTTPDoer) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Adapter{
		client: client,
		now:    time.Now,
	}
}

func (a *Adapter) PutObject(ctx context.Context, cfg oss.ProviderConfig, req oss.PutObjectRequest) (oss.PutObjectResult, error) {
	if err := validateConfig(cfg); err != nil {
		return oss.PutObjectResult{}, err
	}
	if req.Body == nil {
		return oss.PutObjectResult{}, providererror.New(providererror.CategoryValidation, false, "OSS object body is required", nil)
	}
	key, providerKey, err := objectKeys(cfg, req.Key)
	if err != nil {
		return oss.PutObjectResult{}, err
	}
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return oss.PutObjectResult{}, providererror.New(providererror.CategoryTemporary, true, "failed to read OSS object body", err)
	}
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	resp, err := a.do(ctx, http.MethodPut, cfg, providerKey, bytes.NewReader(payload), contentType)
	if err != nil {
		return oss.PutObjectResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return oss.PutObjectResult{}, providerErrorFromResponse(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)

	return oss.PutObjectResult{
		Key:               key,
		ETag:              trimHeaderQuotes(resp.Header.Get("ETag")),
		Size:              int64(len(payload)),
		PublicURL:         publicURL(cfg.PublicBaseURL, providerKey),
		ProviderRequestID: providerRequestID(resp.Header),
	}, nil
}

func (a *Adapter) GetObject(ctx context.Context, cfg oss.ProviderConfig, req oss.GetObjectRequest) (oss.GetObjectResult, error) {
	if err := validateConfig(cfg); err != nil {
		return oss.GetObjectResult{}, err
	}
	key, providerKey, err := objectKeys(cfg, req.Key)
	if err != nil {
		return oss.GetObjectResult{}, err
	}

	resp, err := a.do(ctx, http.MethodGet, cfg, providerKey, nil, "")
	if err != nil {
		return oss.GetObjectResult{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		return oss.GetObjectResult{}, providerErrorFromResponse(resp)
	}

	return oss.GetObjectResult{
		Key:               key,
		Body:              resp.Body,
		ContentType:       resp.Header.Get("Content-Type"),
		Size:              resp.ContentLength,
		ETag:              trimHeaderQuotes(resp.Header.Get("ETag")),
		ProviderRequestID: providerRequestID(resp.Header),
	}, nil
}

func (a *Adapter) DeleteObject(ctx context.Context, cfg oss.ProviderConfig, req oss.DeleteObjectRequest) (oss.DeleteObjectResult, error) {
	if err := validateConfig(cfg); err != nil {
		return oss.DeleteObjectResult{}, err
	}
	_, providerKey, err := objectKeys(cfg, req.Key)
	if err != nil {
		return oss.DeleteObjectResult{}, err
	}

	resp, err := a.do(ctx, http.MethodDelete, cfg, providerKey, nil, "")
	if err != nil {
		return oss.DeleteObjectResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return oss.DeleteObjectResult{Deleted: false, ProviderRequestID: providerRequestID(resp.Header)}, nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return oss.DeleteObjectResult{}, providerErrorFromResponse(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return oss.DeleteObjectResult{Deleted: true, ProviderRequestID: providerRequestID(resp.Header)}, nil
}

func (a *Adapter) PresignObject(_ context.Context, cfg oss.ProviderConfig, req oss.PresignObjectRequest) (oss.PresignObjectResult, error) {
	if err := validateConfig(cfg); err != nil {
		return oss.PresignObjectResult{}, err
	}
	if _, _, err := objectKeys(cfg, req.Key); err != nil {
		return oss.PresignObjectResult{}, err
	}
	return oss.PresignObjectResult{}, providererror.New(providererror.CategoryValidation, false, "OSS presign is not supported", nil)
}

func (a *Adapter) do(ctx context.Context, method string, cfg oss.ProviderConfig, key string, body io.Reader, contentType string) (*http.Response, error) {
	endpoint, err := objectURL(cfg, key)
	if err != nil {
		return nil, err
	}
	payloadHash := emptyPayloadHash()
	if body != nil {
		var payload []byte
		payload, err = io.ReadAll(body)
		if err != nil {
			return nil, providererror.New(providererror.CategoryTemporary, true, "failed to read OSS request body", err)
		}
		payloadHash = sha256HexBytes(payload)
		body = bytes.NewReader(payload)
	}
	if body == nil {
		body = http.NoBody
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, providererror.New(providererror.CategoryValidation, false, "OSS endpoint is invalid", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if err := a.sign(req, cfg, payloadHash); err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, providererror.New(providererror.CategoryTemporary, true, "OSS provider request failed", err)
	}
	return resp, nil
}

func (a *Adapter) sign(req *http.Request, cfg oss.ProviderConfig, payloadHash string) error {
	now := a.now().UTC()
	date := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = defaultRegion
	}

	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("X-Amz-Date", amzDate)

	canonicalHeaders, signedHeaders := canonicalHeaders(req)
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL),
		canonicalQuery(req.URL.Query()),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{date, region, serviceName, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256HexString(canonicalRequest),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(signingKey(cfg.SecretAccessKey, date, region), stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		cfg.AccessKeyID,
		scope,
		signedHeaders,
		signature,
	))
	return nil
}

func validateConfig(cfg oss.ProviderConfig) error {
	if strings.TrimSpace(cfg.EndpointURL) == "" {
		return providererror.New(providererror.CategoryValidation, false, "OSS endpoint URL is required", nil)
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return providererror.New(providererror.CategoryValidation, false, "OSS bucket is required", nil)
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return providererror.New(providererror.CategoryAuth, false, "OSS credential is required", nil)
	}
	return nil
}

func objectKeys(cfg oss.ProviderConfig, rawKey string) (string, string, error) {
	key := cleanObjectKey(rawKey)
	if key == "" {
		return "", "", providererror.New(providererror.CategoryValidation, false, "OSS object key is required", nil)
	}

	prefix := cleanObjectKey(cfg.KeyPrefix)
	if prefix == "" {
		return key, key, nil
	}
	return key, prefix + "/" + key, nil
}

func cleanObjectKey(key string) string {
	key = strings.TrimSpace(strings.ReplaceAll(key, "\\", "/"))
	key = strings.TrimPrefix(key, "/")
	key = filepath.ToSlash(filepath.Clean(key))
	if key == "." || key == ".." || strings.HasPrefix(key, "../") {
		return ""
	}
	return key
}

func objectURL(cfg oss.ProviderConfig, key string) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.EndpointURL), "/"))
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, providererror.New(providererror.CategoryValidation, false, "OSS endpoint URL is invalid", err)
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, providererror.New(providererror.CategoryValidation, false, "OSS endpoint URL is invalid", nil)
	}

	endpoint.Path = joinURLPath(endpoint.Path, cfg.Bucket, key)
	endpoint.RawQuery = ""
	return endpoint, nil
}

func joinURLPath(basePath string, bucket string, key string) string {
	parts := []string{strings.Trim(basePath, "/"), strings.Trim(strings.TrimSpace(bucket), "/"), cleanObjectKey(key)}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return "/" + strings.Join(filtered, "/")
}

func canonicalURI(u *url.URL) string {
	path := u.EscapedPath()
	if path == "" {
		return "/"
	}
	return path
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0)
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		for _, value := range items {
			parts = append(parts, awsEscape(key)+"="+awsEscape(value))
		}
	}
	return strings.Join(parts, "&")
}

func canonicalHeaders(req *http.Request) (string, string) {
	values := map[string]string{
		"host": strings.ToLower(req.URL.Host),
	}
	if contentType := strings.TrimSpace(req.Header.Get("Content-Type")); contentType != "" {
		values["content-type"] = collapseSpaces(contentType)
	}
	values["x-amz-content-sha256"] = strings.TrimSpace(req.Header.Get("X-Amz-Content-Sha256"))
	values["x-amz-date"] = strings.TrimSpace(req.Header.Get("X-Amz-Date"))

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for _, key := range keys {
		canonical.WriteString(key)
		canonical.WriteString(":")
		canonical.WriteString(values[key])
		canonical.WriteString("\n")
	}
	return canonical.String(), strings.Join(keys, ";")
}

func collapseSpaces(value string) string {
	fields := strings.Fields(value)
	return strings.Join(fields, " ")
}

func awsEscape(value string) string {
	escaped := url.QueryEscape(value)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}

func signingKey(secret string, date string, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, serviceName)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func sha256HexString(value string) string {
	return sha256HexBytes([]byte(value))
}

func sha256HexBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func emptyPayloadHash() string {
	return sha256HexBytes(nil)
}

func publicURL(baseURL string, key string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return ""
	}
	segments := strings.Split(cleanObjectKey(key), "/")
	escaped := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(segment))
	}
	return baseURL + "/" + strings.Join(escaped, "/")
}

func providerErrorFromResponse(resp *http.Response) error {
	requestID := providerRequestID(resp.Header)
	category, retryable := categoryForStatus(resp.StatusCode)
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	err := providererror.New(category, retryable, safeMessageForStatus(resp.StatusCode), fmt.Errorf("OSS provider status %d", resp.StatusCode))
	err.ProviderRequestID = requestID
	return err
}

func categoryForStatus(status int) (string, bool) {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return providererror.CategoryAuth, false
	case status == http.StatusTooManyRequests:
		return providererror.CategoryRateLimit, true
	case status == http.StatusRequestTimeout:
		return providererror.CategoryTimeout, true
	case status == http.StatusBadRequest:
		return providererror.CategoryValidation, false
	case status == http.StatusNotFound:
		return providererror.CategoryPermanent, false
	case status >= http.StatusInternalServerError:
		return providererror.CategoryTemporary, true
	default:
		return providererror.CategoryProviderInternal, false
	}
}

func safeMessageForStatus(status int) string {
	switch status {
	case http.StatusNotFound:
		return "OSS object not found"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "OSS provider authentication failed"
	case http.StatusTooManyRequests:
		return "OSS provider rate limit exceeded"
	default:
		return "OSS provider request failed"
	}
}

func providerRequestID(header http.Header) string {
	for _, name := range []string{"X-Amz-Request-Id", "X-Amz-Id-2", "X-Oss-Request-Id", "X-Request-Id"} {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func trimHeaderQuotes(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"`)
}
