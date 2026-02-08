package setup

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultBaseURL = "https://localhost:8080"

type Config struct {
	BaseURL            string
	Token              string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
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
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("unexpected default transport type")
	}

	cloned := transport.Clone()
	cloned.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}

	return &Client{
		baseURL: baseURL,
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: cloned,
		},
	}, nil
}

func (c *Client) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
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
	payload := `{"deleteCollectedGraphData": true,"deleteFileIngestHistory": true,"deleteDataQualityHistory": true,"deleteAssetGroupSelectors": []}`
	req, err := c.newRequest(ctx, http.MethodPost, c.baseURL+"/api/v2/clear-database", strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("reset database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return responseError("reset database", resp)
	}
	return nil
}

func (c *Client) ResetCustomData(ctx context.Context) error {
	if err := c.ResetCustomNodes(ctx); err != nil {
		return err
	}
	if err := c.ResetQueries(ctx); err != nil {
		return err
	}

	if err := c.ResetDatabase(ctx); err != nil {
		return err
	}
	return nil
}
