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
	"strings"
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
	// Prepare URL, preserving query parameters
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Split endpoint and query params to avoid double escaping the query string
	endpointPath := endpoint
	queryString := ""

	if idx := strings.Index(endpoint, "?"); idx >= 0 {
		endpointPath = endpoint[:idx]
		queryString = endpoint[idx+1:]
	}

	// Set path properly without losing query parameters
	u.Path = path.Join(u.Path, "api/v2", endpointPath)

	// Restore or set query string
	if queryString != "" {
		u.RawQuery = queryString
	}

	fullURL := u.String()

	// Log the request details (before making the request)
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	log.Info("REST API Request",
		"requestID", requestID,
		"method", method,
		"url", fullURL)

	// Prepare request body
	var reqBody io.Reader
	var bodyStr string
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyStr = string(jsonBody)
		reqBody = bytes.NewReader(jsonBody)

		// Log request body (if any)
		log.Info("REST API Request Body",
			"requestID", requestID,
			"body", bodyStr)

		// For POST requests, log more details
		if method == http.MethodPost {
			if data, ok := body.(map[string]interface{}); ok {
				log.Info("Creating object with data",
					"requestID", requestID,
					"type", endpoint,
					"name", data["name"])
			}
		}
	}

	// Create request
	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		log.Error(err, "Failed to create HTTP request",
			"requestID", requestID,
			"method", method,
			"url", fullURL)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.SetBasicAuth(c.username, c.password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Log all headers except Authorization (for security)
	headers := make(map[string]string)
	for name, values := range req.Header {
		if name != "Authorization" {
			headers[name] = strings.Join(values, ",")
		}
	}
	log.Info("REST API Request Headers",
		"requestID", requestID,
		"headers", headers)

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		log.Error(err, "REST API Request failed",
			"requestID", requestID,
			"method", method,
			"url", fullURL,
			"duration_ms", requestDuration.Milliseconds())
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "Failed to read response body",
			"requestID", requestID,
			"method", method,
			"url", fullURL)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Always log response status, headers and duration
	respHeaders := make(map[string]string)
	for name, values := range resp.Header {
		respHeaders[name] = strings.Join(values, ",")
	}

	log.Info("REST API Response",
		"requestID", requestID,
		"method", method,
		"url", fullURL,
		"status", resp.StatusCode,
		"statusText", resp.Status,
		"duration_ms", requestDuration.Milliseconds())

	log.Info("REST API Response Headers",
		"requestID", requestID,
		"headers", respHeaders)

	// Log response body - limit size if too large
	respBodyStr := string(respBody)
	if len(respBodyStr) > 1024 {
		// Truncate long responses for logging
		log.Info("REST API Response Body (truncated)",
			"requestID", requestID,
			"bodySize", len(respBodyStr),
			"body", respBodyStr[:1024]+"...")
	} else {
		log.Info("REST API Response Body",
			"requestID", requestID,
			"body", respBodyStr)
	}

	// For POST requests, add additional debug info
	if method == http.MethodPost && resp.StatusCode == http.StatusOK {
		log.Info("POST request successful, analyzing response",
			"requestID", requestID,
			"endpoint", endpoint)

		// Try to quickly check if we got back what we expected
		var resultObj map[string]interface{}
		if err := json.Unmarshal(respBody, &resultObj); err == nil {
			if resultsArray, ok := resultObj["results"].([]interface{}); ok {
				log.Info("Response contains results array",
					"requestID", requestID,
					"count", len(resultsArray))

				// See if any of the results match our request
				if body != nil {
					if data, ok := body.(map[string]interface{}); ok {
						reqName, _ := data["name"].(string)
						if reqName != "" {
							found := false
							for i, item := range resultsArray {
								if obj, ok := item.(map[string]interface{}); ok {
									if name, ok := obj["name"].(string); ok && name == reqName {
										log.Info("Found matching result",
											"requestID", requestID,
											"index", i,
											"name", name)
										found = true
										break
									}
								}
							}
							if !found {
								log.Info("Could not find matching result by name",
									"requestID", requestID,
									"requestedName", reqName,
									"results", len(resultsArray))
							}
						}
					}
				}
			} else {
				// Not a results array, check if it's what we expect
				if name, ok := resultObj["name"].(string); ok {
					log.Info("Response contains direct object",
						"requestID", requestID,
						"name", name)
				}
			}
		}
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Error(nil, "REST API Request failed with error status",
			"requestID", requestID,
			"method", method,
			"url", fullURL,
			"status", resp.StatusCode,
			"response", respBodyStr)
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
	var requestEndpoint string

	// Properly handle URL parameters without escaping the question mark
	if len(filters) > 0 {
		params := url.Values{}
		for key, value := range filters {
			params.Add(key, value)
		}
		// Separate the endpoint from the query string - don't include ? in the endpoint
		requestEndpoint = endpoint

		// The ? will be properly handled by url.Parse in doRequest
		if strings.Contains(requestEndpoint, "?") {
			// If endpoint already has query parameters, append with &
			requestEndpoint = fmt.Sprintf("%s&%s", requestEndpoint, params.Encode())
		} else {
			// Otherwise append with ?
			requestEndpoint = fmt.Sprintf("%s?%s", requestEndpoint, params.Encode())
		}
	} else {
		requestEndpoint = endpoint
	}

	respBody, err := c.doRequest(http.MethodGet, requestEndpoint, nil)
	if err != nil {
		return nil, err
	}

	// First try to parse as a standard paginated response
	var paginatedResult struct {
		Results []map[string]interface{} `json:"results"`
	}
	err = json.Unmarshal(respBody, &paginatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if paginatedResult.Results != nil {
		// Standard paginated response with results array
		// Validate the result objects for required fields
		for i, obj := range paginatedResult.Results {
			if _, ok := obj["id"]; !ok {
				log.Info("API object missing ID field",
					"endpoint", endpoint,
					"index", i,
					"keys", getMapKeys(obj))
			}
		}

		return paginatedResult.Results, nil
	}

	// If no results array found, try parsing as a direct array of objects
	var directResult []map[string]interface{}
	err = json.Unmarshal(respBody, &directResult)
	if err != nil {
		// It's neither a paginated response nor a direct array
		log.Error(err, "Response is neither paginated nor a direct array",
			"endpoint", endpoint)
		return []map[string]interface{}{}, nil
	}

	// Validate the direct result objects for required fields
	for i, obj := range directResult {
		if _, ok := obj["id"]; !ok {
			log.Info("API object missing ID field in direct array",
				"endpoint", endpoint,
				"index", i,
				"keys", getMapKeys(obj))
		}
	}

	return directResult, nil
}

// CreateObject creates an object in the AWX API
func (c *Client) CreateObject(endpoint string, data map[string]interface{}) (map[string]interface{}, error) {
	respBody, err := c.doRequest(http.MethodPost, endpoint, data)
	if err != nil {
		return nil, err
	}

	// First try to parse as a direct object
	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if we already have a valid created object
	if id, hasID := result["id"]; hasID {
		log.Info("API returned direct object with ID",
			"endpoint", endpoint,
			"id", id)
		return result, nil
	}

	// Check if the response has a 'results' array - some AWX endpoints return collections
	if results, ok := result["results"].([]interface{}); ok && len(results) > 0 {
		log.Info("API returned results array instead of direct object",
			"endpoint", endpoint,
			"count", len(results))

		// Get the name from the request data for matching
		objectName, hasName := data["name"].(string)

		// First look for an exact match by name if we have a name in the request
		if hasName {
			for _, item := range results {
				if obj, ok := item.(map[string]interface{}); ok {
					if name, ok := obj["name"].(string); ok && name == objectName {
						log.Info("Found exact match for created object by name",
							"endpoint", endpoint,
							"name", objectName)
						return obj, nil
					}
				}
			}

			// If we couldn't find a match by name, log it
			log.Error(nil, "Could not find created object in results",
				"endpoint", endpoint,
				"name", objectName)
		}

		// If no match was found and we have results, let's try a different approach
		// The issue might be that the API is returning a list of existing objects, not the created one
		// Let's do a separate find request to get the object by name
		if hasName {
			log.Info("Trying to find created object with separate request",
				"endpoint", endpoint,
				"name", objectName)

			// Use FindObjectByName to get the created object
			filters := map[string]string{"name": objectName}
			objects, findErr := c.ListObjects(endpoint, filters)
			if findErr != nil {
				log.Error(findErr, "Failed to find created object",
					"endpoint", endpoint,
					"name", objectName)
			} else if len(objects) > 0 {
				log.Info("Found created object with separate request",
					"endpoint", endpoint,
					"name", objectName)
				return objects[0], nil
			}
		}

		// If all else fails, take the first object from the original results
		if len(results) > 0 {
			if resultObj, ok := results[0].(map[string]interface{}); ok {
				log.Info("Using first object from results as fallback",
					"endpoint", endpoint)
				return resultObj, nil
			}
		}

		// This is a serious issue - can't find the object we tried to create
		log.Error(nil, "Failed to extract object from results array",
			"endpoint", endpoint)
		return nil, fmt.Errorf("could not find created object in API response")
	}

	// If we get here, the response format was unexpected
	log.Error(nil, "Unexpected API response format",
		"endpoint", endpoint,
		"keys", getMapKeys(result))
	return nil, fmt.Errorf("unexpected API response format")
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

	// Verify the object has an ID field
	result := objects[0]
	if _, ok := result["id"]; !ok {
		log.Error(nil, "Object returned by API missing ID field",
			"endpoint", endpoint,
			"name", name,
			"keys", getMapKeys(result))
	}

	return result, nil
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
