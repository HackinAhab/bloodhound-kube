package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func (c *Client) customNodesURL() string {
	return c.baseURL + "/api/v2/custom-nodes"
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
