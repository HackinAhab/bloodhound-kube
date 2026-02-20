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

	req, err := c.newRequest(ctx, http.MethodPost, c.customNodesURL(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Upload model request failed", "error", err)
		return errors.New("upload model failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := responseError("upload model", resp)
		c.log.Error("Upload model failed", "status", resp.StatusCode, "error", err)
		return errors.New("upload model failed")
	}

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
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Delete custom node request failed", "node", nodeName, "error", err)
		return errors.New("delete custom node failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := responseError(fmt.Sprintf("delete custom node %q", nodeName), resp)
		c.log.Error("Delete custom node failed", "node", nodeName, "status", resp.StatusCode, "error", err)
		return errors.New("delete custom node failed")
	}

	c.log.Debug("Custom node deleted", "node", nodeName)

	return nil
}

func (c *Client) GetCustomNodes(ctx context.Context) ([]CustomNode, error) {
	c.log.Debug("Fetching custom nodes")
	req, err := c.newRequest(ctx, http.MethodGet, c.customNodesURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Get custom nodes request failed", "error", err)
		return nil, errors.New("get custom nodes failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := responseError("get custom nodes", resp)
		c.log.Error("Get custom nodes failed", "status", resp.StatusCode, "error", err)
		return nil, errors.New("get custom nodes failed")
	}

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
