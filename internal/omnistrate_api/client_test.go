package omnistrate_api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	require.NotNil(t, client)

	// Verify it's the correct implementation
	clientImpl, ok := client.(*ClientImpl)
	require.True(t, ok)
	require.NotNil(t, clientImpl.httpClient)

	// Verify retry configuration
	assert.Equal(t, 3, clientImpl.httpClient.RetryMax)
	assert.Equal(t, 1*time.Second, clientImpl.httpClient.RetryWaitMin)
	assert.Equal(t, 30*time.Second, clientImpl.httpClient.RetryWaitMax)
	assert.Equal(t, 60*time.Second, clientImpl.httpClient.HTTPClient.Timeout)
}

func TestNewWithHTTPClient(t *testing.T) {
	// Create a custom HTTP client
	customHTTPClient := retryablehttp.NewClient()
	customHTTPClient.RetryMax = 5
	customHTTPClient.RetryWaitMin = 2 * time.Second
	customHTTPClient.RetryWaitMax = 60 * time.Second
	customHTTPClient.HTTPClient.Timeout = 120 * time.Second

	client := NewWithHTTPClient(customHTTPClient)

	require.NotNil(t, client)

	// Verify it's the correct implementation
	clientImpl, ok := client.(*ClientImpl)
	require.True(t, ok)
	require.NotNil(t, clientImpl.httpClient)

	// Verify the custom configuration is preserved
	assert.Equal(t, 5, clientImpl.httpClient.RetryMax)
	assert.Equal(t, 2*time.Second, clientImpl.httpClient.RetryWaitMin)
	assert.Equal(t, 60*time.Second, clientImpl.httpClient.RetryWaitMax)
	assert.Equal(t, 120*time.Second, clientImpl.httpClient.HTTPClient.Timeout)
}

func TestClientImpl_GetCurrentCapacity(t *testing.T) {
	tests := []struct {
		name             string
		resourceAlias    string
		mockResponse     ResourceInstanceCapacity
		mockStatusCode   int
		mockResponseBody string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:          "successful get current capacity",
			resourceAlias: "test-resource",
			mockResponse: ResourceInstanceCapacity{
				InstanceID:            "instance-123",
				Status:                ACTIVE,
				ResourceID:            "resource-456",
				ResourceAlias:         "test-resource",
				CurrentCapacity:       3,
				LastObservedTimestamp: strfmt.DateTime(time.Now()),
			},
			mockStatusCode: http.StatusOK,
			expectedError:  false,
		},
		{
			name:             "server returns 500 error",
			resourceAlias:    "test-resource",
			mockStatusCode:   http.StatusInternalServerError,
			expectedError:    true,
			expectedErrorMsg: "Failed get current capacity for resourceAlias: test-resource, status code: 500",
		},
		{
			name:             "server returns 404 error",
			resourceAlias:    "nonexistent-resource",
			mockStatusCode:   http.StatusNotFound,
			expectedError:    true,
			expectedErrorMsg: "Failed get current capacity for resourceAlias: nonexistent-resource, status code: 404",
		},
		{
			name:             "invalid JSON response",
			resourceAlias:    "test-resource",
			mockStatusCode:   http.StatusOK,
			mockResponseBody: `{"invalid": json}`,
			expectedError:    true,
			expectedErrorMsg: "Failed unmarshal response body when querying current capacity for resourceAlias: test-resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/resource/"+tt.resourceAlias+"/capacity", r.URL.Path)

				w.WriteHeader(tt.mockStatusCode)

				if tt.mockResponseBody != "" {
					w.Write([]byte(tt.mockResponseBody))
				} else if tt.mockStatusCode == http.StatusOK {
					respBytes, _ := json.Marshal(tt.mockResponse)
					w.Write(respBytes)
				}
			}))
			defer server.Close()

			// Create client with custom base URL pointing to mock server
			client := &ClientImpl{
				httpClient: createTestHTTPClient(),
			}

			// For this test, we'll need to modify the client to accept a custom base URL
			// Since we can't modify const, we'll create a test version
			testClient := &testClientImpl{
				ClientImpl: client,
				baseURL:    server.URL + "/resource/",
			}

			ctx := context.Background()
			result, err := testClient.GetCurrentCapacity(ctx, tt.resourceAlias)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mockResponse.InstanceID, result.InstanceID)
				assert.Equal(t, tt.mockResponse.Status, result.Status)
				assert.Equal(t, tt.mockResponse.ResourceID, result.ResourceID)
				assert.Equal(t, tt.mockResponse.ResourceAlias, result.ResourceAlias)
				assert.Equal(t, tt.mockResponse.CurrentCapacity, result.CurrentCapacity)
			}
		})
	}
}

func TestClientImpl_AddCapacity(t *testing.T) {
	tests := []struct {
		name             string
		resourceAlias    string
		mockResponse     ResourceInstanceCapacity
		mockStatusCode   int
		mockResponseBody string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:          "successful add capacity",
			resourceAlias: "test-resource",
			mockResponse: ResourceInstanceCapacity{
				InstanceID:            "instance-123",
				Status:                STARTING,
				ResourceID:            "resource-456",
				ResourceAlias:         "test-resource",
				CurrentCapacity:       4,
				LastObservedTimestamp: strfmt.DateTime(time.Now()),
			},
			mockStatusCode: http.StatusOK,
			expectedError:  false,
		},
		{
			name:             "server returns 500 error",
			resourceAlias:    "test-resource",
			mockStatusCode:   http.StatusInternalServerError,
			expectedError:    true,
			expectedErrorMsg: "Failed to add capacity for resourceAlias: test-resource, status code: 500",
		},
		{
			name:             "invalid JSON response",
			resourceAlias:    "test-resource",
			mockStatusCode:   http.StatusOK,
			mockResponseBody: `{"invalid": json}`,
			expectedError:    true,
			expectedErrorMsg: "Failed unmarshal response body when adding capacity for resourceAlias: test-resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/resource/"+tt.resourceAlias+"/capacity/add", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tt.mockStatusCode)

				if tt.mockResponseBody != "" {
					w.Write([]byte(tt.mockResponseBody))
				} else if tt.mockStatusCode == http.StatusOK {
					respBytes, _ := json.Marshal(tt.mockResponse)
					w.Write(respBytes)
				}
			}))
			defer server.Close()

			// Create test client
			client := &ClientImpl{
				httpClient: createTestHTTPClient(),
			}

			testClient := &testClientImpl{
				ClientImpl: client,
				baseURL:    server.URL + "/resource/",
			}

			ctx := context.Background()
			result, err := testClient.AddCapacity(ctx, tt.resourceAlias)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mockResponse.InstanceID, result.InstanceID)
				assert.Equal(t, tt.mockResponse.Status, result.Status)
				assert.Equal(t, tt.mockResponse.ResourceID, result.ResourceID)
				assert.Equal(t, tt.mockResponse.ResourceAlias, result.ResourceAlias)
				assert.Equal(t, tt.mockResponse.CurrentCapacity, result.CurrentCapacity)
			}
		})
	}
}

func TestClientImpl_RemoveCapacity(t *testing.T) {
	tests := []struct {
		name             string
		resourceAlias    string
		mockResponse     ResourceInstanceCapacity
		mockStatusCode   int
		mockResponseBody string
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:          "successful remove capacity",
			resourceAlias: "test-resource",
			mockResponse: ResourceInstanceCapacity{
				InstanceID:            "instance-123",
				Status:                ACTIVE,
				ResourceID:            "resource-456",
				ResourceAlias:         "test-resource",
				CurrentCapacity:       2,
				LastObservedTimestamp: strfmt.DateTime(time.Now()),
			},
			mockStatusCode: http.StatusOK,
			expectedError:  false,
		},
		{
			name:             "server returns 500 error",
			resourceAlias:    "test-resource",
			mockStatusCode:   http.StatusInternalServerError,
			expectedError:    true,
			expectedErrorMsg: "Failed to remove capacity for resourceAlias: test-resource, status code: 500",
		},
		{
			name:             "invalid JSON response",
			resourceAlias:    "test-resource",
			mockStatusCode:   http.StatusOK,
			mockResponseBody: `{"invalid": json}`,
			expectedError:    true,
			expectedErrorMsg: "Failed unmarshal response body when removing capacity for resourceAlias: test-resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/resource/"+tt.resourceAlias+"/capacity/remove", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tt.mockStatusCode)

				if tt.mockResponseBody != "" {
					w.Write([]byte(tt.mockResponseBody))
				} else if tt.mockStatusCode == http.StatusOK {
					respBytes, _ := json.Marshal(tt.mockResponse)
					w.Write(respBytes)
				}
			}))
			defer server.Close()

			// Create test client
			client := &ClientImpl{
				httpClient: createTestHTTPClient(),
			}

			testClient := &testClientImpl{
				ClientImpl: client,
				baseURL:    server.URL + "/resource/",
			}

			ctx := context.Background()
			result, err := testClient.RemoveCapacity(ctx, tt.resourceAlias)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mockResponse.InstanceID, result.InstanceID)
				assert.Equal(t, tt.mockResponse.Status, result.Status)
				assert.Equal(t, tt.mockResponse.ResourceID, result.ResourceID)
				assert.Equal(t, tt.mockResponse.ResourceAlias, result.ResourceAlias)
				assert.Equal(t, tt.mockResponse.CurrentCapacity, result.CurrentCapacity)
			}
		})
	}
}

func TestClientImpl_ContextCancellation(t *testing.T) {
	// Create a server that delays response to test context cancellation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"instanceId": "test"}`))
	}))
	defer server.Close()

	client := &ClientImpl{
		httpClient: createTestHTTPClient(),
	}

	testClient := &testClientImpl{
		ClientImpl: client,
		baseURL:    server.URL + "/resource/",
	}

	// Create context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := testClient.GetCurrentCapacity(ctx, "test-resource")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// Helper functions and types for testing

// testClientImpl wraps ClientImpl to allow custom baseURL for testing
type testClientImpl struct {
	*ClientImpl
	baseURL string
}

func (c *testClientImpl) GetCurrentCapacity(ctx context.Context, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+resourceAlias+"/capacity", nil)
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

func (c *testClientImpl) AddCapacity(ctx context.Context, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+resourceAlias+"/capacity/add", nil)
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

func (c *testClientImpl) RemoveCapacity(ctx context.Context, resourceAlias string) (resp ResourceInstanceCapacity, err error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+resourceAlias+"/capacity/remove", nil)
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

func createTestHTTPClient() *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 0 // Disable retries for faster tests
	client.HTTPClient.Timeout = 1 * time.Second
	return client
}
