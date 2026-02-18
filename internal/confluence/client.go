package confluence

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// Cloud API paths
	v2Path = "/api/v2"  // Relative to base URL (which includes /wiki for Cloud)
	v1Path = "/rest/api" // Relative to base URL

	maxRetries = 3
)

// Client is the Confluence REST API client (supports Cloud and Server/DC).
type Client struct {
	baseURL      string // e.g. "https://company.atlassian.net/wiki" or "https://confluence.company.com"
	authHeader   string
	httpClient   *http.Client
	instanceType InstanceType

	// spaceKeyCache maps space key -> space ID for v2 operations.
	spaceKeyCache map[string]string
}

// NewClient creates a new Confluence API client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("confluence: base URL is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("confluence: token is required")
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	authType := cfg.AuthType
	if authType == "" {
		if cfg.Email != "" {
			authType = AuthBasic
		} else {
			authType = AuthBearer
		}
	}

	var authHeader string
	switch authType {
	case AuthBearer:
		authHeader = "Bearer " + cfg.Token
	default: // AuthBasic
		if cfg.Email == "" {
			return nil, fmt.Errorf("confluence: email is required for basic auth")
		}
		creds := cfg.Email + ":" + cfg.Token
		authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	if cfg.InsecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return &Client{
		baseURL:       baseURL,
		authHeader:    authHeader,
		httpClient:    httpClient,
		instanceType:  cfg.InstanceType,
		spaceKeyCache: make(map[string]string),
	}, nil
}

// IsCloud returns true if this is a Confluence Cloud instance.
func (c *Client) IsCloud() bool {
	return c.instanceType == InstanceCloud
}

// BaseURL returns the base URL of the Confluence instance.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// SetHTTPClient overrides the default HTTP client (useful for testing).
func (c *Client) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}

// --- Path builders ---

// v2URL builds a full URL for v2 API (Cloud only).
func (c *Client) v2URL(segments ...string) string {
	return c.baseURL + v2Path + "/" + strings.Join(segments, "/")
}

// v1URL builds a full URL for v1 API.
func (c *Client) v1URL(segments ...string) string {
	return c.baseURL + v1Path + "/" + strings.Join(segments, "/")
}

// --- Space key resolver ---

// ResolveSpaceKey translates a space key to a space ID (v2 uses numeric IDs).
// Results are cached in memory.
func (c *Client) ResolveSpaceKey(key string) (string, error) {
	if id, ok := c.spaceKeyCache[key]; ok {
		return id, nil
	}

	q := url.Values{"keys": {key}}
	data, err := c.getV2("spaces", q)
	if err != nil {
		return "", fmt.Errorf("resolving space key %q: %w", key, err)
	}

	var page CursorPage[Space]
	if err := json.Unmarshal(data, &page); err != nil {
		return "", fmt.Errorf("parsing space response: %w", err)
	}

	if len(page.Results) == 0 {
		return "", fmt.Errorf("space %q not found", key)
	}

	id := page.Results[0].ID
	c.spaceKeyCache[key] = id
	return id, nil
}

// --- Convenience methods (versioned) ---

// getV2 performs a GET request to the v2 API.
func (c *Client) getV2(path string, query url.Values) ([]byte, error) {
	fullURL := c.v2URL(path)
	return c.request(http.MethodGet, fullURL, query, nil)
}

// getV1 performs a GET request to the v1 API.
func (c *Client) getV1(path string, query url.Values) ([]byte, error) {
	fullURL := c.v1URL(path)
	return c.request(http.MethodGet, fullURL, query, nil)
}

// postV2 performs a POST request to the v2 API.
func (c *Client) postV2(path string, body interface{}) ([]byte, error) {
	fullURL := c.v2URL(path)
	return c.request(http.MethodPost, fullURL, nil, body)
}

// putV2 performs a PUT request to the v2 API.
func (c *Client) putV2(path string, body interface{}) ([]byte, error) {
	fullURL := c.v2URL(path)
	return c.request(http.MethodPut, fullURL, nil, body)
}

// deleteV2 performs a DELETE request to the v2 API.
func (c *Client) deleteV2(path string) ([]byte, error) {
	fullURL := c.v2URL(path)
	return c.request(http.MethodDelete, fullURL, nil, nil)
}

// Get performs a raw GET request (for custom paths).
func (c *Client) Get(fullPath string, query url.Values) ([]byte, error) {
	fullURL := c.baseURL + fullPath
	return c.request(http.MethodGet, fullURL, query, nil)
}

// --- Internal HTTP helpers ---

func (c *Client) request(method, fullURL string, query url.Values, body interface{}) ([]byte, error) {
	if query != nil {
		fullURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("confluence: failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(method, fullURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("confluence: failed to create request: %w", err)
		}

		req.Header.Set("Authorization", c.authHeader)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("confluence: request failed: %w", err)
			if isNetworkError(err) {
				return nil, fmt.Errorf("%w\n\nHint: could not reach %s — check your network connection or corporate VPN", lastErr, c.baseURL)
			}
			if attempt < maxRetries {
				time.Sleep(backoff(attempt))
				if bodyBytes != nil {
					bodyReader = bytes.NewReader(bodyBytes)
				}
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("confluence: failed to read response body: %w", err)
		}

		// Success.
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// Rate limited — retry after backoff.
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = parseAPIError(resp.StatusCode, respBody)
			if attempt < maxRetries {
				time.Sleep(backoff(attempt))
				if bodyBytes != nil {
					bodyReader = bytes.NewReader(bodyBytes)
				}
				continue
			}
			return nil, lastErr
		}

		// Server errors — retry with backoff.
		if resp.StatusCode >= 500 {
			lastErr = parseAPIError(resp.StatusCode, respBody)
			if attempt < maxRetries {
				time.Sleep(backoff(attempt))
				if bodyBytes != nil {
					bodyReader = bytes.NewReader(bodyBytes)
				}
				continue
			}
			return nil, lastErr
		}

		// Client errors (4xx) — don't retry.
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	return nil, lastErr
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > 60*time.Second {
		d = 60 * time.Second
	}
	return d
}

func parseAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode}
	if err := json.Unmarshal(body, apiErr); err != nil {
		apiErr.Message = fmt.Sprintf("HTTP %d: %s", statusCode, string(body))
	}
	if apiErr.Message == "" && apiErr.ErrorMessage == "" {
		apiErr.Message = fmt.Sprintf("HTTP %d", statusCode)
	}
	return apiErr
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var dnsErr *net.DNSError
	var opErr *net.OpError
	if errors.As(err, &dnsErr) {
		return true
	}
	if errors.As(err, &opErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "i/o timeout")
}
