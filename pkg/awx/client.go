package awx

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
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

// TestConnection tests the connection to the AWX instance
func (c *Client) TestConnection() error {
	// Make a request to the /api/ endpoint to check if the connection works
	endpoint := fmt.Sprintf("%s/api/", c.baseURL)

	// Log the connection attempt
	log.Info("Testing connection to AWX", "url", endpoint)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Error(err, "Failed to create request", "url", endpoint)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	req.SetBasicAuth(c.username, c.password)

	// Create a client with appropriate TLS configuration based on the protocol
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// If using HTTPS, configure TLS
	if u, err := url.Parse(c.baseURL); err == nil && u.Scheme == "https" {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // For testing; consider proper TLS verification in production
			},
		}
	}

	// Perform the request with detailed error handling
	resp, err := client.Do(req)
	if err != nil {
		// Log connection error details
		log.Error(err, "Connection to AWX failed",
			"url", endpoint,
			"baseURL", c.baseURL,
			"username", c.username)

		// Check for common network errors and provide more context
		if urlErr, ok := err.(*url.Error); ok {
			if urlErr.Timeout() {
				return fmt.Errorf("connection timeout: %w", err)
			} else if opErr, ok := urlErr.Err.(*net.OpError); ok {
				if opErr.Op == "dial" {
					return fmt.Errorf("cannot reach host (dns or network issue): %w", err)
				} else if opErr.Op == "read" {
					return fmt.Errorf("connection reset or closed by host: %w", err)
				}
			}
		}
		return fmt.Errorf("failed to connect to AWX: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body for error details if status is not OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyContent := string(bodyBytes)

		log.Error(nil, "Unexpected status code from AWX",
			"statusCode", resp.StatusCode,
			"url", endpoint,
			"response", bodyContent)

		// Add specific error messages for common status codes
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401): %s", bodyContent)
		case http.StatusForbidden:
			return fmt.Errorf("permission denied (403): %s", bodyContent)
		case http.StatusNotFound:
			return fmt.Errorf("API endpoint not found (404): %s", bodyContent)
		case http.StatusServiceUnavailable:
			return fmt.Errorf("service unavailable (503): %s", bodyContent)
		default:
			return fmt.Errorf("unexpected status code: %d - %s", resp.StatusCode, bodyContent)
		}
	}

	log.Info("Successfully connected to AWX", "url", endpoint)
	return nil
}
