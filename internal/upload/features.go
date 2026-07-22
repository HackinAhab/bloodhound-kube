package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// opengraphExtensionManagementKey is the stable feature flag key for enabling
// OpenGraph management extensions. Feature flag IDs are not stable across
// BloodHound versions/instances, so we always resolve the ID via this key
// rather than hardcoding it.
const opengraphExtensionManagementKey = "opengraph_extension_management"

type featureFlag struct {
	ID      int    `json:"id"`
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

func (c *Client) featuresURL() string {
	return c.baseURL + "/api/v2/features"
}

func (c *Client) featureToggleURL(id int) string {
	return fmt.Sprintf("%s/%d/toggle", c.featuresURL(), id)
}

func (c *Client) getFeatureFlags(ctx context.Context) ([]featureFlag, error) {
	c.log.Debug("Fetching feature flags")
	resp, err := c.doRequest(ctx, http.MethodGet, c.featuresURL(), "get feature flags", nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload struct {
		Data []featureFlag `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode feature flags response: %w", err)
	}

	c.log.Debug("Fetched feature flags", "count", len(payload.Data))
	return payload.Data, nil
}

// EnableOpenGraphExtensions enables the OpenGraph extension management
// feature flag, identified by key (not ID, since IDs are not stable across
// instances). It is a no-op if the flag is already enabled. The returned
// bool reports whether the flag was actually toggled (false if it was
// already enabled).
func (c *Client) EnableOpenGraphExtensions(ctx context.Context) (bool, error) {
	flags, err := c.getFeatureFlags(ctx)
	if err != nil {
		return false, fmt.Errorf("fetch feature flags: %w", err)
	}

	var flag *featureFlag
	for i := range flags {
		if flags[i].Key == opengraphExtensionManagementKey {
			flag = &flags[i]
			break
		}
	}
	if flag == nil {
		return false, fmt.Errorf("feature flag %q not found", opengraphExtensionManagementKey)
	}

	if flag.Enabled {
		c.log.Info("OpenGraph extension management already enabled")
		return false, nil
	}

	c.log.Info("Enabling OpenGraph extension management", "id", flag.ID)
	resp, err := c.doRequest(ctx, http.MethodPut, c.featureToggleURL(flag.ID), "toggle opengraph extension management", nil, http.StatusOK)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var toggled struct {
		Data struct {
			Enabled bool `json:"enabled"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&toggled); err != nil {
		return false, fmt.Errorf("decode toggle response: %w", err)
	}
	if !toggled.Data.Enabled {
		return false, errors.New("opengraph extension management toggle did not report enabled")
	}

	c.log.Info("OpenGraph extension management enabled")
	return true, nil
}
