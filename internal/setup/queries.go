package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

func (c *Client) savedQueriesImportURL() string {
	return c.baseURL + "/api/v2/saved-queries/import"
}

func (c *Client) savedQueriesURL() string {
	return c.baseURL + "/api/v2/saved-queries"
}

func (c *Client) UploadQueriesFromFile(ctx context.Context, queriesFile string) (int, error) {
	c.log.Info("Uploading queries", "file", queriesFile)
	data, err := os.ReadFile(queriesFile)
	if err != nil {
		c.log.Error("Read queries file failed", "file", queriesFile, "error", err)
		return 0, errors.New("upload queries failed")
	}

	var config QueriesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		c.log.Error("Invalid queries JSON", "file", queriesFile, "error", err)
		return 0, errors.New("upload queries failed")
	}

	if len(config.Queries) == 0 {
		c.log.Error("Queries file contains no queries", "file", queriesFile)
		return 0, errors.New("upload queries failed")
	}

	c.log.Info("Uploading queries from file", "count", len(config.Queries))

	for i, query := range config.Queries {
		payload, err := json.Marshal(query)
		if err != nil {
			c.log.Error("Marshal query failed", "index", i+1, "error", err)
			return i, errors.New("upload queries failed")
		}
		c.log.Debug("Uploading query", "index", i+1)
		if err := c.uploadQuery(ctx, payload); err != nil {
			return i, err
		}
	}

	c.log.Info("Queries upload completed", "count", len(config.Queries))

	return len(config.Queries), nil
}

func (c *Client) uploadQuery(ctx context.Context, payload []byte) error {
	req, err := c.newRequest(ctx, http.MethodPost, c.savedQueriesImportURL(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Upload query request failed", "error", err)
		return errors.New("upload queries failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		err := responseError("upload query", resp)
		c.log.Error("Upload query failed", "status", resp.StatusCode, "error", err)
		return errors.New("upload queries failed")
	}

	return nil
}

func (c *Client) deleteQuery(ctx context.Context, queryID string) error {
	if queryID == "" {
		c.log.Error("Query ID is required")
		return errors.New("delete query failed")
	}

	c.log.Debug("Deleting query", "query_id", queryID)
	url := c.savedQueriesURL() + "/" + queryID
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Delete query request failed", "query_id", queryID, "error", err)
		return errors.New("delete query failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		err := responseError(fmt.Sprintf("delete query %q", queryID), resp)
		c.log.Error("Delete query failed", "query_id", queryID, "status", resp.StatusCode, "error", err)
		return errors.New("delete query failed")
	}

	c.log.Debug("Query deleted", "query_id", queryID)

	return nil
}

func (c *Client) getQueries(ctx context.Context) ([]map[string]any, error) {
	c.log.Debug("Fetching saved queries")
	req, err := c.newRequest(ctx, http.MethodGet, c.savedQueriesURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("Get queries request failed", "error", err)
		return nil, errors.New("get queries failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := responseError("get queries", resp)
		c.log.Error("Get queries failed", "status", resp.StatusCode, "error", err)
		return nil, errors.New("get queries failed")
	}

	var payload struct {
		Data []map[string]any `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		c.log.Error("Decode queries response failed", "error", err)
		return nil, errors.New("get queries failed")
	}

	if payload.Data == nil {
		c.log.Error("Queries response missing data")
		return nil, errors.New("get queries failed")
	}

	c.log.Debug("Fetched saved queries", "count", len(payload.Data))
	return payload.Data, nil
}

func (c *Client) ResetQueries(ctx context.Context) error {
	c.log.Info("Resetting saved queries")
	queries, err := c.getQueries(ctx)
	if err != nil {
		c.log.Error("Fetch existing queries failed", "error", err)
		return errors.New("reset queries failed")
	}

	for i, query := range queries {
		idValue, ok := query["id"]
		if !ok || idValue == nil {
			c.log.Error("Query missing valid ID", "index", i+1)
			return errors.New("reset queries failed")
		}
		idNumber, ok := idValue.(json.Number)
		if !ok {
			c.log.Error("Query missing numeric ID", "index", i+1)
			return errors.New("reset queries failed")
		}
		id, err := idNumber.Int64()
		if err != nil {
			c.log.Error("Query has invalid ID", "index", i+1, "error", err)
			return errors.New("reset queries failed")
		}
		idString := fmt.Sprintf("%d", id)
		if err := c.deleteQuery(ctx, idString); err != nil {
			c.log.Error("Delete query failed", "query_id", idString, "error", err)
			return errors.New("reset queries failed")
		}
	}

	c.log.Info("Saved queries reset completed")

	return nil
}
