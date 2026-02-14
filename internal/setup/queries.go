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

func (c *Client) savedQueriesImportURL() string {
	return c.baseURL + "/api/v2/saved-queries/import"
}

func (c *Client) savedQueriesURL() string {
	return c.baseURL + "/api/v2/saved-queries"
}

func (c *Client) UploadQueriesFromFile(ctx context.Context, queriesFile string) (int, error) {
	c.log.WithField("file", queriesFile).Info("Uploading queries")
	data, err := os.ReadFile(queriesFile)
	if err != nil {
		c.log.WithError(err).WithField("file", queriesFile).Error("Read queries file failed")
		return 0, errors.New("upload queries failed")
	}

	var config QueriesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		c.log.WithError(err).WithField("file", queriesFile).Error("Invalid queries JSON")
		return 0, errors.New("upload queries failed")
	}

	if len(config.Queries) == 0 {
		c.log.WithField("file", queriesFile).Error("Queries file contains no queries")
		return 0, errors.New("upload queries failed")
	}

	c.log.WithField("count", len(config.Queries)).Info("Uploading queries from file")

	for i, query := range config.Queries {
		payload, err := json.Marshal(query)
		if err != nil {
			c.log.WithError(err).WithField("index", i+1).Error("Marshal query failed")
			return i, errors.New("upload queries failed")
		}
		c.log.WithField("index", i+1).Debug("Uploading query")
		if err := c.uploadQuery(ctx, payload); err != nil {
			return i, err
		}
	}

	c.log.WithField("count", len(config.Queries)).Info("Queries upload completed")

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
		c.log.WithError(err).Error("Upload query request failed")
		return errors.New("upload queries failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		err := responseError("upload query", resp)
		c.log.WithError(err).WithField("status", resp.StatusCode).Error("Upload query failed")
		return errors.New("upload queries failed")
	}

	return nil
}

func (c *Client) deleteQuery(ctx context.Context, queryID string) error {
	if queryID == "" {
		c.log.Error("Query ID is required")
		return errors.New("delete query failed")
	}

	c.log.WithField("query_id", queryID).Debug("Deleting query")
	url := c.savedQueriesURL() + "/" + queryID
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.WithError(err).WithField("query_id", queryID).Error("Delete query request failed")
		return errors.New("delete query failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		err := responseError(fmt.Sprintf("delete query %q", queryID), resp)
		c.log.WithError(err).WithFields(logrus.Fields{"query_id": queryID, "status": resp.StatusCode}).Error("Delete query failed")
		return errors.New("delete query failed")
	}

	c.log.WithField("query_id", queryID).Debug("Query deleted")

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
		c.log.WithError(err).Error("Get queries request failed")
		return nil, errors.New("get queries failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := responseError("get queries", resp)
		c.log.WithError(err).WithField("status", resp.StatusCode).Error("Get queries failed")
		return nil, errors.New("get queries failed")
	}

	var payload struct {
		Data []map[string]any `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		c.log.WithError(err).Error("Decode queries response failed")
		return nil, errors.New("get queries failed")
	}

	if payload.Data == nil {
		c.log.Error("Queries response missing data")
		return nil, errors.New("get queries failed")
	}

	c.log.WithField("count", len(payload.Data)).Debug("Fetched saved queries")
	return payload.Data, nil
}

func (c *Client) ResetQueries(ctx context.Context) error {
	c.log.Info("Resetting saved queries")
	queries, err := c.getQueries(ctx)
	if err != nil {
		c.log.WithError(err).Error("Fetch existing queries failed")
		return errors.New("reset queries failed")
	}

	for i, query := range queries {
		idValue, ok := query["id"]
		if !ok || idValue == nil {
			c.log.WithField("index", i+1).Error("Query missing valid ID")
			return errors.New("reset queries failed")
		}
		idNumber, ok := idValue.(json.Number)
		if !ok {
			c.log.WithField("index", i+1).Error("Query missing numeric ID")
			return errors.New("reset queries failed")
		}
		id, err := idNumber.Int64()
		if err != nil {
			c.log.WithError(err).WithField("index", i+1).Error("Query has invalid ID")
			return errors.New("reset queries failed")
		}
		idString := fmt.Sprintf("%d", id)
		if err := c.deleteQuery(ctx, idString); err != nil {
			c.log.WithError(err).WithField("query_id", idString).Error("Delete query failed")
			return errors.New("reset queries failed")
		}
	}

	c.log.Info("Saved queries reset completed")

	return nil
}
