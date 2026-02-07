package setup

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

func (c *Client) UploadModel(ctx context.Context, modelFile string) error {
	data, err := os.ReadFile(modelFile)
	if err != nil {
		return fmt.Errorf("read model file %q: %w", modelFile, err)
	}

	if err := validateJSON(data); err != nil {
		return fmt.Errorf("invalid model JSON %q: %w", modelFile, err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, c.customNodesURL(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return responseError("upload model", resp)
	}

	return nil
}

func (c *Client) DeleteCustomNode(ctx context.Context, nodeName string) error {
	if nodeName == "" {
		return fmt.Errorf("node name is required")
	}

	url := fmt.Sprintf("%s/%s", c.customNodesURL(), nodeName)
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete custom node %q: %w", nodeName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return responseError(fmt.Sprintf("delete custom node %q", nodeName), resp)
	}

	return nil
}

func (c *Client) GetCustomNodes(ctx context.Context) ([]CustomNode, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.customNodesURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get custom nodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, responseError("get custom nodes", resp)
	}

	var payload customNodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode custom nodes response: %w", err)
	}

	if payload.Data == nil {
		return nil, fmt.Errorf("custom nodes response missing data")
	}

	return payload.Data, nil
}

func (c *Client) ResetCustomNodes(ctx context.Context) error {
	nodes, err := c.GetCustomNodes(ctx)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if err := c.DeleteCustomNode(ctx, node.KindName); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) UploadQueriesFromFile(ctx context.Context, queriesFile string) (int, error) {
	data, err := os.ReadFile(queriesFile)
	if err != nil {
		return 0, fmt.Errorf("read queries file %q: %w", queriesFile, err)
	}

	var config QueriesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, fmt.Errorf("invalid queries JSON %q: %w", queriesFile, err)
	}

	if len(config.Queries) == 0 {
		return 0, fmt.Errorf("queries file %q does not contain any queries", queriesFile)
	}

	for i, query := range config.Queries {
		payload, err := json.Marshal(query)
		if err != nil {
			return i, fmt.Errorf("marshal query %d: %w", i+1, err)
		}
		if err := c.uploadQuery(ctx, payload); err != nil {
			return i, err
		}
	}

	return len(config.Queries), nil
}

func (c *Client) customNodesURL() string {
	return c.baseURL + "/api/v2/custom-nodes"
}

func (c *Client) savedQueriesImportURL() string {
	return c.baseURL + "/api/v2/saved-queries/import"
}

func (c *Client) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	return req, nil
}

func (c *Client) uploadQuery(ctx context.Context, payload []byte) error {
	req, err := c.newRequest(ctx, http.MethodPost, c.savedQueriesImportURL(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return responseError("upload query", resp)
	}

	return nil
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
