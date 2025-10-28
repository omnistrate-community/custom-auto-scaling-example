package omnistrate_api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

const (
	baseURL = "http://127.0.0.1:49750/resource/"
)

type Client interface {
	GetCurrentCapacity(ctx context.Context, resourceAlias string) (ResourceInstanceCapacity, error)
	AddCapacity(ctx context.Context, resourceAlias string) (ResourceInstanceCapacity, error)
	RemoveCapacity(ctx context.Context, resourceAlias string) (ResourceInstanceCapacity, error)
}

/**
 * This file contains all APIs used to interact with omnistrate platform via local sidecar.
 */
type ClientImpl struct {
	httpClient *retryablehttp.Client
}

func NewClient() Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 30 * time.Second
	retryClient.HTTPClient.Timeout = 60 * time.Second
	return &ClientImpl{retryClient}
}

func (c *ClientImpl) GetCurrentCapacity(ctx context.Context, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, baseURL+resourceAlias+"/capacity", nil)
	if err != nil {
		return
	}
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "Failed get current capacity for resourceAlias: %s", resourceAlias)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		err = errors.Errorf("Failed get current capacity for resourceAlias: %s, status code: %d", resourceAlias, httpResp.StatusCode)
		return
	}
	defer func() {
		if closeErr := httpResp.Body.Close(); closeErr != nil {
			err = errors.Wrapf(closeErr, "Failed to close response body")
		}
	}()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		err = errors.Wrapf(err, "Failed read response body when querying current capacity for resourceAlias: %s", resourceAlias)
		return
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = errors.Wrapf(err, "Failed unmarshal response body when querying current capacity for resourceAlias: %s", resourceAlias)
		return
	}
	return
}

func (c *ClientImpl) AddCapacity(ctx context.Context, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, baseURL+resourceAlias+"/capacity/add", nil)
	if err != nil {
		return ResourceInstanceCapacity{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "Failed to add capacity for resourceAlias: %s", resourceAlias)
		return ResourceInstanceCapacity{}, err
	}
	if httpResp.StatusCode != http.StatusOK {
		err = errors.Errorf("Failed to add capacity for resourceAlias: %s, status code: %d", resourceAlias, httpResp.StatusCode)
		return ResourceInstanceCapacity{}, err
	}
	defer func() {
		if closeErr := httpResp.Body.Close(); closeErr != nil {
			err = errors.Wrapf(closeErr, "Failed to close response body")
		}
	}()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		err = errors.Wrapf(err, "Failed read response body when adding capacity for resourceAlias: %s", resourceAlias)
		return ResourceInstanceCapacity{}, err
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = errors.Wrapf(err, "Failed unmarshal response body when adding capacity for resourceAlias: %s", resourceAlias)
		return ResourceInstanceCapacity{}, err
	}
	return resp, nil
}

func (c *ClientImpl) RemoveCapacity(ctx context.Context, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, baseURL+resourceAlias+"/capacity/remove", nil)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create remove capacity request for resourceAlias: %s", resourceAlias)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "Failed to remove capacity for resourceAlias: %s", resourceAlias)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		err = errors.Errorf("Failed to remove capacity for resourceAlias: %s, status code: %d", resourceAlias, httpResp.StatusCode)
		return
	}
	defer func() {
		if closeErr := httpResp.Body.Close(); closeErr != nil {
			err = errors.Wrapf(closeErr, "Failed to close response body")
		}
	}()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		err = errors.Wrapf(err, "Failed read response body when removing capacity for resourceAlias: %s", resourceAlias)
		return
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err = errors.Wrapf(err, "Failed unmarshal response body when removing capacity for resourceAlias: %s", resourceAlias)
		return
	}
	return resp, nil
}
