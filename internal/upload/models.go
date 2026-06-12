package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

func (c *Client) customNodesURL() string {
	return c.baseURL + "/api/v2/custom-nodes"
}

func (c *Client) UploadModel(ctx context.Context, modelFile string) error {
	c.log.Info("Uploading model", "file", modelFile)
	data, err := os.ReadFile(modelFile)
	if err != nil {
		return fmt.Errorf("read model file %s: %w", modelFile, err)
	}

	if err := validateJSON(data); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", modelFile, err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, c.customNodesURL(), "upload model", bytes.NewReader(data), http.StatusCreated)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.log.Info("Model upload completed", "file", modelFile)

	return nil
}

func (c *Client) DeleteCustomNode(ctx context.Context, nodeName string) error {
	if nodeName == "" {
		return errors.New("custom node name is required")
	}

	c.log.Debug("Deleting custom node", "node", nodeName)

	url := fmt.Sprintf("%s/%s", c.customNodesURL(), nodeName)
	resp, err := c.doRequest(ctx, http.MethodDelete, url, "delete custom node", nil, http.StatusOK)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.log.Debug("Custom node deleted", "node", nodeName)

	return nil
}

func (c *Client) GetCustomNodes(ctx context.Context) ([]CustomNode, error) {
	c.log.Debug("Fetching custom nodes")
	resp, err := c.doRequest(ctx, http.MethodGet, c.customNodesURL(), "get custom nodes", nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload customNodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode custom nodes response: %w", err)
	}

	if payload.Data == nil {
		return nil, errors.New("custom nodes response missing data")
	}

	c.log.Debug("Fetched custom nodes", "count", len(payload.Data))
	return payload.Data, nil
}

func (c *Client) ResetCustomNodes(ctx context.Context) error {
	c.log.Info("Resetting custom nodes")
	nodes, err := c.GetCustomNodes(ctx)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if err := c.DeleteCustomNode(ctx, node.KindName); err != nil {
			return err
		}
	}

	c.log.Info("Custom nodes reset completed")

	return nil
}
