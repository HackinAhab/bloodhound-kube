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

func (c *Client) uploadJobCreateURL() string {
	return c.baseURL + "/api/v2/file-upload/start"
}

func (c *Client) uploadFiletoJobURL(jobID string) string {
	return c.baseURL + "/api/v2/file-upload/" + jobID
}

func (c *Client) uploadJobCompleteURL(jobID string) string {
	return c.baseURL + "/api/v2/file-upload/" + jobID + "/end"
}

func (c *Client) createUploadJob(ctx context.Context) (string, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, c.uploadJobCreateURL(), "upload collections", nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload struct {
		Data struct {
			ID json.Number `json:"id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return "", fmt.Errorf("decode upload job response: %w", err)
	}
	jobID := payload.Data.ID.String()
	if jobID == "" {
		return "", errors.New("upload job response missing id")
	}

	return jobID, nil
}

func (c *Client) uploadFileToJob(ctx context.Context, jobID string, data []byte) error {
	resp, err := c.doRequest(ctx, http.MethodPost, c.uploadFiletoJobURL(jobID), "upload collections", bytes.NewReader(data), http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (c *Client) completeUploadJob(ctx context.Context, jobID string) error {
	resp, err := c.doRequest(ctx, http.MethodPost, c.uploadJobCompleteURL(jobID), "upload collections", nil, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) UploadOutput(ctx context.Context, outputFile string) error {
	c.log.Info("Uploading collections", "file", outputFile)
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return fmt.Errorf("read collections file %s: %w", outputFile, err)
	}

	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON in %s", outputFile)
	}

	jobID, err := c.createUploadJob(ctx)
	if err != nil {
		return err
	}

	c.log.Info("Collections upload job created", "job_id", jobID)

	completed := false
	defer func() {
		if completed {
			return
		}
		if err := c.completeUploadJob(ctx, jobID); err != nil {
			c.log.Error("Failed to complete upload job after error", "job_id", jobID, "error", err)
		}
	}()

	if err := c.uploadFileToJob(ctx, jobID, data); err != nil {
		return err
	}

	if err := c.completeUploadJob(ctx, jobID); err != nil {
		return err
	}
	completed = true

	c.log.Info("Collections upload completed", "file", outputFile, "job_id", jobID)

	return nil
}
