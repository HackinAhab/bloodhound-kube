package setup

import (
	"bytes"
	"context"
	"encoding/json"
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

func (c *Client) deleteQuery(ctx context.Context, queryID string) error {
	if queryID == "" {
		return fmt.Errorf("query ID is required")
	}
	url := c.savedQueriesURL() + "/" + queryID
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete query %q: %w", queryID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return responseError(fmt.Sprintf("delete query %q", queryID), resp)
	}

	return nil
}

func (c *Client) getQueries(ctx context.Context) ([]map[string]any, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.savedQueriesURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get queries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, responseError("get queries", resp)
	}

	var payload struct {
		Data []map[string]any `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode queries response: %w", err)
	}

	if payload.Data == nil {
		return nil, fmt.Errorf("queries response missing data")
	}

	return payload.Data, nil
}

func (c *Client) ResetQueries(ctx context.Context) error {
	queries, err := c.getQueries(ctx)
	if err != nil {
		return fmt.Errorf("fetch existing queries: %w", err)
	}

	for i, query := range queries {
		idValue, ok := query["id"]
		if !ok || idValue == nil {
			return fmt.Errorf("query %d missing valid ID", i+1)
		}
		idNumber, ok := idValue.(json.Number)
		if !ok {
			return fmt.Errorf("query %d missing numeric ID", i+1)
		}
		id, err := idNumber.Int64()
		if err != nil {
			return fmt.Errorf("query %d has invalid ID: %w", i+1, err)
		}
		idString := fmt.Sprintf("%d", id)
		if err := c.deleteQuery(ctx, idString); err != nil {
			return fmt.Errorf("delete query %q: %w", idString, err)
		}
	}

	return nil
}
