package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	template "text/template"
)

func (c *Client) savedQueriesImportURL() string {
	return c.baseURL + "/api/v2/saved-queries/import"
}

func (c *Client) savedQueriesURL() string {
	return c.baseURL + "/api/v2/saved-queries"
}

func (c *Client) UploadQueriesFromFile(ctx context.Context, queriesFile string, clusterName string) (int, error) {
	c.log.Info("Uploading queries", "file", queriesFile)
	data, err := os.ReadFile(queriesFile)
	if err != nil {
		return 0, fmt.Errorf("read queries file %s: %w", queriesFile, err)
	}

	var config QueriesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, fmt.Errorf("invalid queries JSON in %s: %w", queriesFile, err)
	}

	if len(config.Queries) == 0 {
		return 0, fmt.Errorf("queries file %s contains no queries", queriesFile)
	}

	if err := renderQueryTemplates(config.Queries, clusterName); err != nil {
		return 0, err
	}

	c.log.Info("Uploading queries from file", "count", len(config.Queries))

	for i, query := range config.Queries {
		payload, err := json.Marshal(query)
		if err != nil {
			return i, fmt.Errorf("marshal query %d: %w", i+1, err)
		}
		c.log.Debug("Uploading query", "index", i+1)
		if err := c.uploadQuery(ctx, payload); err != nil {
			return i, err
		}
	}

	c.log.Info("Queries upload completed", "count", len(config.Queries))

	return len(config.Queries), nil
}

type queryTemplateData struct {
	Cluster string
}

func renderQueryTemplates(queries []map[string]any, clusterName string) error {
	cluster := strings.TrimSpace(clusterName)
	if cluster == "" {
		cluster = "default"
	}
	data := queryTemplateData{Cluster: cluster}
	for i, q := range queries {
		raw, ok := q["query"].(string)
		if !ok {
			continue
		}
		tmpl, err := template.New("").Parse(raw)
		if err != nil {
			return fmt.Errorf("query %d has invalid template: %w", i+1, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("query %d template execution failed: %w", i+1, err)
		}
		queries[i]["query"] = buf.String()
	}
	return nil
}

func (c *Client) uploadQuery(ctx context.Context, payload []byte) error {
	resp, err := c.doRequest(ctx, http.MethodPost, c.savedQueriesImportURL(), "upload query", bytes.NewReader(payload), http.StatusOK, http.StatusCreated)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) deleteQuery(ctx context.Context, queryID string) error {
	if queryID == "" {
		return errors.New("query ID is required")
	}

	c.log.Debug("Deleting query", "query_id", queryID)
	url := c.savedQueriesURL() + "/" + queryID
	resp, err := c.doRequest(ctx, http.MethodDelete, url, "delete query", nil, http.StatusNoContent)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.log.Debug("Query deleted", "query_id", queryID)

	return nil
}

func (c *Client) getQueries(ctx context.Context) ([]map[string]any, error) {
	c.log.Debug("Fetching saved queries")
	resp, err := c.doRequest(ctx, http.MethodGet, c.savedQueriesURL(), "get queries", nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload struct {
		Data []map[string]any `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode queries response: %w", err)
	}

	if payload.Data == nil {
		return nil, errors.New("queries response missing data")
	}

	c.log.Debug("Fetched saved queries", "count", len(payload.Data))
	return payload.Data, nil
}

func (c *Client) ResetQueries(ctx context.Context) error {
	c.log.Info("Resetting saved queries")
	queries, err := c.getQueries(ctx)
	if err != nil {
		return fmt.Errorf("fetch existing queries: %w", err)
	}

	for i, query := range queries {
		idValue, ok := query["id"]
		if !ok || idValue == nil {
			return fmt.Errorf("query %d missing id", i+1)
		}
		idNumber, ok := idValue.(json.Number)
		if !ok {
			return fmt.Errorf("query %d has non-numeric id", i+1)
		}
		id, err := idNumber.Int64()
		if err != nil {
			return fmt.Errorf("query %d invalid id: %w", i+1, err)
		}
		idString := fmt.Sprintf("%d", id)
		if err := c.deleteQuery(ctx, idString); err != nil {
			return err
		}
	}

	c.log.Info("Saved queries reset completed")

	return nil
}
