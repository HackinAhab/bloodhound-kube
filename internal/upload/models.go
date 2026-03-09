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
		c.log.Error("Read model file failed", "file", modelFile, "error", err)
		return errors.New("upload model failed")
	}

	if err := validateJSON(data); err != nil {
		c.log.Error("Invalid model JSON", "file", modelFile, "error", err)
		return errors.New("upload model failed")
	}

	resp, err := c.doRequest(ctx, http.MethodPost, c.customNodesURL(), "upload model", bytes.NewReader(data), http.StatusCreated)
	if err != nil {
		return errors.New("upload model failed")
	}
	defer resp.Body.Close()

	c.log.Info("Model upload completed", "file", modelFile)

	return nil
}

func (c *Client) DeleteCustomNode(ctx context.Context, nodeName string) error {
	if nodeName == "" {
		c.log.Error("Custom node name is required")
		return errors.New("delete custom node failed")
	}

	c.log.Debug("Deleting custom node", "node", nodeName)

	url := fmt.Sprintf("%s/%s", c.customNodesURL(), nodeName)
	resp, err := c.doRequest(ctx, http.MethodDelete, url, "delete custom node", nil, http.StatusOK)
	if err != nil {
		return errors.New("delete custom node failed")
	}
	defer resp.Body.Close()

	c.log.Debug("Custom node deleted", "node", nodeName)

	return nil
}

func (c *Client) GetCustomNodes(ctx context.Context) ([]CustomNode, error) {
	c.log.Debug("Fetching custom nodes")
	resp, err := c.doRequest(ctx, http.MethodGet, c.customNodesURL(), "get custom nodes", nil, http.StatusOK)
	if err != nil {
		return nil, errors.New("get custom nodes failed")
	}
	defer resp.Body.Close()

	var payload customNodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		c.log.Error("Decode custom nodes response failed", "error", err)
		return nil, errors.New("get custom nodes failed")
	}

	if payload.Data == nil {
		c.log.Error("Custom nodes response missing data")
		return nil, errors.New("get custom nodes failed")
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
