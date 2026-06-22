package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func (c *Client) extensionsURL() string {
	return c.baseURL + "/api/v2/extensions"
}

func (c *Client) UploadExtension(ctx context.Context, modelFile string) error {
	c.log.Info("Uploading extension schema", "file", modelFile)
	data, err := os.ReadFile(modelFile)
	if err != nil {
		return fmt.Errorf("read model file %s: %w", modelFile, err)
	}

	if err := validateJSON(data); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", modelFile, err)
	}

	resp, err := c.doRequest(ctx, http.MethodPut, c.extensionsURL(), "upload extension", bytes.NewReader(data), http.StatusOK, http.StatusCreated)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.log.Info("Extension schema upload completed", "file", modelFile)
	return nil
}

func (c *Client) DeleteExtension(ctx context.Context, extensionID int) error {
	c.log.Debug("Deleting extension", "id", extensionID)

	url := fmt.Sprintf("%s/%d", c.extensionsURL(), extensionID)
	resp, err := c.doRequest(ctx, http.MethodDelete, url, "delete extension", nil, http.StatusNoContent)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.log.Debug("Extension deleted", "id", extensionID)
	return nil
}

func (c *Client) GetExtensions(ctx context.Context) ([]Extension, error) {
	c.log.Debug("Fetching extensions")
	resp, err := c.doRequest(ctx, http.MethodGet, c.extensionsURL(), "get extensions", nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload extensionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode extensions response: %w", err)
	}

	c.log.Debug("Fetched extensions", "count", len(payload.Data.Extensions))
	return payload.Data.Extensions, nil
}

func (c *Client) ResetExtensions(ctx context.Context) error {
	c.log.Info("Resetting extensions")
	extensions, err := c.GetExtensions(ctx)
	if err != nil {
		return err
	}

	for _, ext := range extensions {
		if ext.IsBuiltin {
			continue
		}
		if err := c.DeleteExtension(ctx, ext.ID); err != nil {
			return err
		}
	}

	c.log.Info("Extensions reset completed")
	return nil
}
