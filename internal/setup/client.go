package setup

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
		logger.Error("Token ID and token key are required")
		return nil, errors.New("setup client initialization failed")
	}

	auth, err := sdk.NewSecurityProviderHMACCredentials(cfg.TokenKey, cfg.TokenID)
	if err != nil {
		logger.Error("Failed to initialize HMAC credentials", "error", err)
		return nil, errors.New("setup client initialization failed")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		logger.Error("Unexpected default transport type")
		return nil, errors.New("setup client initialization failed")
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
		c.log.Error("Create request failed", "method", method, "url", url, "error", err)
		return nil, errors.New("setup request failed")
	}
	if c.auth == nil {
		c.log.Error("HMAC credentials not configured")
		return nil, errors.New("setup request failed")
	}
	if err := c.auth.Intercept(ctx, req); err != nil {
		c.log.Error("Authenticate request failed", "method", method, "url", url, "error", err)
		return nil, errors.New("setup request failed")
	}
	return req, nil
}

func validateJSON(data []byte) error {
	var payload any
	return json.Unmarshal(data, &payload)
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

func (c *Client) ResetDatabase(ctx context.Context) error {
	c.log.Info("Resetting database")
	payload := `{"deleteCollectedGraphData": true,"deleteFileIngestHistory": true,"deleteDataQualityHistory": true,"deleteAssetGroupSelectors": []}`
	req, err := c.newRequest(ctx, http.MethodPost, c.baseURL+"/api/v2/clear-database", strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Reset database request failed", "error", err)
		return errors.New("reset database failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		err := responseError("reset database", resp)
		c.log.Error("Reset database failed", "status", resp.StatusCode, "error", err)
		return errors.New("reset database failed")
	}

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
