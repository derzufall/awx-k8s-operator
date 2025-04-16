package awx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("awx-client")

// Client represents an AWX API client
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewClient creates a new AWX API client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request to the AWX API
func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	// Prepare URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path.Join(u.Path, "api/v2", endpoint)

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.SetBasicAuth(c.username, c.password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetObject retrieves an object from the AWX API
func (c *Client) GetObject(endpoint string, id int) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%d/", endpoint, id)
	respBody, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// ListObjects lists objects from the AWX API with optional filters
func (c *Client) ListObjects(endpoint string, filters map[string]string) ([]map[string]interface{}, error) {
	urlWithParams := endpoint
	if len(filters) > 0 {
		params := url.Values{}
		for key, value := range filters {
			params.Add(key, value)
		}
		urlWithParams = fmt.Sprintf("%s/?%s", endpoint, params.Encode())
	}

	respBody, err := c.doRequest(http.MethodGet, urlWithParams, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []map[string]interface{} `json:"results"`
	}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Results, nil
}

// CreateObject creates an object in the AWX API
func (c *Client) CreateObject(endpoint string, data map[string]interface{}) (map[string]interface{}, error) {
	respBody, err := c.doRequest(http.MethodPost, endpoint, data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// UpdateObject updates an object in the AWX API
func (c *Client) UpdateObject(endpoint string, id int, data map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%d/", endpoint, id)
	respBody, err := c.doRequest(http.MethodPatch, url, data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// DeleteObject deletes an object from the AWX API
func (c *Client) DeleteObject(endpoint string, id int) error {
	url := fmt.Sprintf("%s/%d/", endpoint, id)
	_, err := c.doRequest(http.MethodDelete, url, nil)
	return err
}

// FindObjectByName finds an object by name in the AWX API
func (c *Client) FindObjectByName(endpoint, name string) (map[string]interface{}, error) {
	filters := map[string]string{"name": name}
	objects, err := c.ListObjects(endpoint, filters)
	if err != nil {
		return nil, err
	}

	if len(objects) == 0 {
		return nil, nil
	}

	return objects[0], nil
}
