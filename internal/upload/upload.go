package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	req, err := c.newRequest(ctx, http.MethodPost, c.uploadJobCreateURL(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Failed to create upload job", "error", err)
		return "", errors.New("upload collections failed")
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		err := responseError("upload collections", resp)
		c.log.Error("Upload job creation failed", "status", resp.StatusCode, "error", err)
		resp.Body.Close()
		return "", errors.New("upload collections failed")
	}

	var payload struct {
		Data struct {
			ID json.Number `json:"id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		c.log.Error("Decode upload job response failed", "error", err)
		resp.Body.Close()
		return "", errors.New("upload collections failed")
	}
	resp.Body.Close()
	jobID := payload.Data.ID.String()
	if jobID == "" {
		c.log.Error("Upload job response missing id")
		return "", errors.New("upload collections failed")
	}

	return jobID, nil
}

func (c *Client) uploadFileToJob(ctx context.Context, jobID string, data []byte) error {
	req, err := c.newRequest(ctx, http.MethodPost, c.uploadFiletoJobURL(jobID), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Upload collections request failed", "job_id", jobID, "error", err)
		return errors.New("upload collections failed")
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := responseError("upload collections", resp)
		c.log.Error("Upload collections failed", "job_id", jobID, "status", resp.StatusCode, "error", err)
		resp.Body.Close()
		return errors.New("upload collections failed")
	}
	resp.Body.Close()

	return nil
}

func (c *Client) completeUploadJob(ctx context.Context, jobID string) error {
	req, err := c.newRequest(ctx, http.MethodPost, c.uploadJobCompleteURL(jobID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Complete upload job request failed", "job_id", jobID, "error", err)
		return errors.New("upload collections failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		err := responseError("upload collections", resp)
		c.log.Error("Complete upload job failed", "job_id", jobID, "status", resp.StatusCode, "error", err)
		return errors.New("upload collections failed")
	}

	return nil
}

func (c *Client) UploadOutput(ctx context.Context, outputFile string) error {
	c.log.Info("Uploading collections", "file", outputFile)
	data, err := os.ReadFile(outputFile)
	if err != nil {
		c.log.Error("Read collections file failed", "file", outputFile, "error", err)
		return errors.New("upload collections failed")
	}

	if err := validateJSON(data); err != nil {
		c.log.Error("Invalid Parsed JSON", "file", outputFile, "error", err)
		return errors.New("upload collections failed")
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
