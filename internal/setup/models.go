package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

func (c *Client) customNodesURL() string {
	return c.baseURL + "/api/v2/custom-nodes"
}

func (c *Client) UploadModel(ctx context.Context, modelFile string) error {
	c.log.WithField("file", modelFile).Info("Uploading model")
	data, err := os.ReadFile(modelFile)
	if err != nil {
		c.log.WithError(err).WithField("file", modelFile).Error("Read model file failed")
		return errors.New("upload model failed")
	}

	if err := validateJSON(data); err != nil {
		c.log.WithError(err).WithField("file", modelFile).Error("Invalid model JSON")
		return errors.New("upload model failed")
	}

	req, err := c.newRequest(ctx, http.MethodPost, c.customNodesURL(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.WithError(err).Error("Upload model request failed")
		return errors.New("upload model failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := responseError("upload model", resp)
		c.log.WithError(err).WithField("status", resp.StatusCode).Error("Upload model failed")
		return errors.New("upload model failed")
	}

	c.log.WithField("file", modelFile).Info("Model upload completed")

	return nil
}

func (c *Client) DeleteCustomNode(ctx context.Context, nodeName string) error {
	if nodeName == "" {
		c.log.Error("Custom node name is required")
		return errors.New("delete custom node failed")
	}

	c.log.WithField("node", nodeName).Debug("Deleting custom node")

	url := fmt.Sprintf("%s/%s", c.customNodesURL(), nodeName)
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.WithError(err).WithField("node", nodeName).Error("Delete custom node request failed")
		return errors.New("delete custom node failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := responseError(fmt.Sprintf("delete custom node %q", nodeName), resp)
		c.log.WithError(err).WithFields(logrus.Fields{"node": nodeName, "status": resp.StatusCode}).Error("Delete custom node failed")
		return errors.New("delete custom node failed")
	}

	c.log.WithField("node", nodeName).Debug("Custom node deleted")

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
		c.log.WithError(err).Error("Get custom nodes request failed")
		return nil, errors.New("get custom nodes failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := responseError("get custom nodes", resp)
		c.log.WithError(err).WithField("status", resp.StatusCode).Error("Get custom nodes failed")
		return nil, errors.New("get custom nodes failed")
	}

	var payload customNodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		c.log.WithError(err).Error("Decode custom nodes response failed")
		return nil, errors.New("get custom nodes failed")
	}

	if payload.Data == nil {
		c.log.Error("Custom nodes response missing data")
		return nil, errors.New("get custom nodes failed")
	}

	c.log.WithField("count", len(payload.Data)).Debug("Fetched custom nodes")
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
