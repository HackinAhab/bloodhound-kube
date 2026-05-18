package upload

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"bloodhound-kube/internal/utils"

	"github.com/SpecterOps/bloodhound-go-sdk/sdk"
)

const DefaultBaseURL = "https://localhost:8080"

type Config struct {
	BaseURL            string
	TokenID            string
	TokenKey           string
	InsecureSkipVerify bool
	Timeout            time.Duration
	Logger             utils.Logger
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       *sdk.HMACCredentials
	log        utils.Logger
}

type CustomNode struct {
	KindName string `json:"kindName"`
}

type QueriesConfig struct {
	Queries []map[string]any `json:"queries"`
}

type customNodesResponse struct {
	Data []CustomNode `json:"data"`
}

func NewClient(cfg Config) (*Client, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = utils.New("info", false)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	if cfg.TokenID == "" || cfg.TokenKey == "" {
		return nil, errors.New("token ID and token key are required")
	}

	auth, err := sdk.NewSecurityProviderHMACCredentials(cfg.TokenKey, cfg.TokenID)
	if err != nil {
		return nil, fmt.Errorf("initialize HMAC credentials: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("unexpected default transport type")
	}

	cloned := transport.Clone()
	cloned.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: cloned,
		},
		auth: auth,
		log:  logger,
	}, nil
}

func (c *Client) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.auth == nil {
		return nil, errors.New("HMAC credentials not configured")
	}
	if err := c.auth.Intercept(ctx, req); err != nil {
		return nil, fmt.Errorf("authenticate request: %w", err)
	}
	return req, nil
}

func validateJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	seenToken := false
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		seenToken = true
	}
	if !seenToken {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func responseError(action string, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s failed with status %d", action, resp.StatusCode)
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Errorf("%s failed with status %d", action, resp.StatusCode)
	}

	return fmt.Errorf("%s failed with status %d: %s", action, resp.StatusCode, trimmed)
}

func statusAllowed(status int, allowed []int) bool {
	return slices.Contains(allowed, status)
}

func (c *Client) doRequest(ctx context.Context, method, url, action string, body io.Reader, expectedStatus ...int) (*http.Response, error) {
	req, err := c.newRequest(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error(action+" request failed", "method", method, "url", url, "error", err)
		return nil, fmt.Errorf("%s: %w", action, err)
	}

	if !statusAllowed(resp.StatusCode, expectedStatus) {
		err := responseError(action, resp)
		c.log.Error(action+" failed", "method", method, "url", url, "status", resp.StatusCode, "error", err)
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}

func (c *Client) ResetDatabase(ctx context.Context) error {
	c.log.Info("Resetting database")
	payload := `{"deleteCollectedGraphData": true,"deleteFileIngestHistory": true,"deleteDataQualityHistory": true,"deleteAssetGroupSelectors": []}`
	resp, err := c.doRequest(ctx, http.MethodPost, c.baseURL+"/api/v2/clear-database", "reset database", strings.NewReader(payload), http.StatusNoContent)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.log.Info("Database reset completed")
	return nil
}

func (c *Client) ResetCustomData(ctx context.Context) error {
	c.log.Info("Resetting custom data")
	if err := c.ResetCustomNodes(ctx); err != nil {
		return err
	}
	if err := c.ResetQueries(ctx); err != nil {
		return err
	}

	if err := c.ResetDatabase(ctx); err != nil {
		return err
	}

	c.log.Info("Custom data reset completed")
	return nil
}
