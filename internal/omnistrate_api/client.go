package omnistrate_api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	baseURL = "http://127.0.0.1:49750/resource/"
)

/**
 * This file contains all APIs used to interact with omnistrate platform via local sidecar.
 */
type Client struct {
	httpClient *http.Client
}

func NewClientWithContext(ctx context.Context) *Client {
	return &Client{&http.Client{Timeout: 60 * time.Second, Transport: http.DefaultTransport}}
}

func (c *Client) GetCurrentCapacity(ctx context.Context, instanceId, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+resourceAlias+"/capacity", nil)
	if err != nil {
		return
	}
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "Failed get current capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		err = errors.Errorf("Failed get current capacity for instance %s. resourceAlias: %s, status code: %d", instanceId, resourceAlias, httpResp.StatusCode)
		return
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		err = errors.Wrapf(err, "Failed read response body when querying current capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = errors.Wrapf(err, "Failed unmarshal response body when querying current capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return
	}
	return
}

func (c *Client) AddCapacity(ctx context.Context, instanceId, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+resourceAlias+"/capacity/add", nil)
	if err != nil {
		return ResourceInstanceCapacity{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "Failed to add capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return ResourceInstanceCapacity{}, err
	}
	if httpResp.StatusCode != http.StatusOK {
		err = errors.Errorf("Failed to add capacity for instance %s. resourceAlias: %s, status code: %d", instanceId, resourceAlias, httpResp.StatusCode)
		return ResourceInstanceCapacity{}, err
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		err = errors.Wrapf(err, "Failed read response body when adding capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return ResourceInstanceCapacity{}, err
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = errors.Wrapf(err, "Failed unmarshal response body when adding capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return ResourceInstanceCapacity{}, err
	}
	return resp, nil
}

func (c *Client) RemoveCapacity(ctx context.Context, instanceId, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+resourceAlias+"/capacity/remove", nil)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create remove capacity request for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "Failed to remove capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		err = errors.Errorf("Failed to remove capacity for instance %s. resourceAlias: %s, status code: %d", instanceId, resourceAlias, httpResp.StatusCode)
		return
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		err = errors.Wrapf(err, "Failed read response body when removing capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = errors.Wrapf(err, "Failed unmarshal response body when removing capacity for instance %s. resourceAlias: %s", instanceId, resourceAlias)
		return
	}
	return resp, nil
}
