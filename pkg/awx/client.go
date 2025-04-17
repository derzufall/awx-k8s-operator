package awx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	// Verify the response has an ID field
	if _, ok := result["id"]; !ok {
		log.Error(nil, "Object returned by API missing ID field",
			"endpoint", endpoint,
			"id", id,
			"keys", getMapKeys(result))
		return nil, fmt.Errorf("API returned object without ID field")
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

	// First try to parse as a standard paginated response (most common in AWX)
	var paginatedResult struct {
		Count    int                      `json:"count"`
		Next     *string                  `json:"next"`
		Previous *string                  `json:"previous"`
		Results  []map[string]interface{} `json:"results"`
	}
	err = json.Unmarshal(respBody, &paginatedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if paginatedResult.Results != nil {
		// Standard paginated response with results array (AWX's typical format)
		log.Info("API returned paginated response",
			"endpoint", endpoint,
			"count", paginatedResult.Count,
			"resultsCount", len(paginatedResult.Results))

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
		// Neither a paginated response nor a direct array - log error and return empty array
		log.Error(err, "Response is neither paginated nor a direct array",
			"endpoint", endpoint)
		return []map[string]interface{}{}, nil
	}

	log.Info("API returned direct array",
		"endpoint", endpoint,
		"count", len(directResult))

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

// Post performs a POST request to the AWX API
func (c *Client) Post(endpoint string, body interface{}) (*http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Set path properly
	u.Path = path.Join(u.Path, "api/v2", endpoint)
	fullURL := u.String()

	// Marshal request body
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	reqBody := bytes.NewReader(jsonBody)

	// Create request
	req, err := http.NewRequest(http.MethodPost, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	return c.httpClient.Do(req)
}

// GetObjectByName retrieves an object from the AWX API by name
func (c *Client) GetObjectByName(endpoint, name string) (map[string]interface{}, error) {
	return c.FindObjectByName(endpoint, name)
}

// CreateObject creates an object in the AWX API
func (c *Client) CreateObject(endpoint string, payload map[string]interface{}, expectedObj string) (map[string]interface{}, error) {
	// Check if the object exists first
	name, hasName := payload["name"]
	if hasName {
		log.Info("Checking if object exists before creation", "endpoint", endpoint, "name", name)
		existing, err := c.GetObjectByName(endpoint, name.(string))
		if err == nil && existing != nil {
			log.Info("Object already exists, returning existing object", "endpoint", endpoint, "name", name)
			return existing, nil
		}
	}

	log.Info("Creating object", "endpoint", endpoint, "keys", getMapKeys(payload))
	resp, err := c.Post(endpoint, payload)
	if err != nil {
		log.Error(err, "Failed to create object", "endpoint", endpoint)
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Error(nil, "Error response from AWX API",
			"status", resp.Status,
			"endpoint", endpoint,
			"response", string(body))
		return nil, fmt.Errorf("failed to create object: %s", resp.Status)
	}

	result := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Error(err, "Failed to decode response", "endpoint", endpoint)
		return nil, err
	}

	log.Info("Received response", "endpoint", endpoint, "status", resp.Status, "keys", getMapKeys(result))

	// Handle the case where the API returns a collection instead of a direct object
	if results, ok := result["results"].([]interface{}); ok {
		log.Info("API returned a collection", "endpoint", endpoint, "count", len(results))

		// Try to find our newly created object in the results
		if hasName {
			nameStr := payload["name"].(string)
			for _, item := range results {
				if obj, ok := item.(map[string]interface{}); ok {
					if objName, ok := obj["name"].(string); ok && objName == nameStr {
						log.Info("Found newly created object in results", "endpoint", endpoint, "name", nameStr)
						return obj, nil
					}
				}
			}
			log.Error(nil, "Failed to find newly created object in results",
				"endpoint", endpoint,
				"name", nameStr,
				"result_count", len(results))
			return nil, fmt.Errorf("object creation failed: object not found in response")
		}

		// If we don't have a name to search for, just return the result as is
		return result, nil
	}

	// Check if the result has id, if not it's probably an error
	if _, hasID := result["id"]; !hasID {
		if hasName {
			nameStr := payload["name"].(string)
			log.Error(nil, "Failed to create object: response missing ID",
				"endpoint", endpoint,
				"name", nameStr,
				"keys", getMapKeys(result))

			// Verify if object was actually created despite missing ID in response
			created, err := c.GetObjectByName(endpoint, nameStr)
			if err == nil && created != nil {
				log.Info("Object was actually created despite missing ID in response",
					"endpoint", endpoint,
					"name", nameStr)
				return created, nil
			}

			return nil, fmt.Errorf("object creation failed: response missing ID")
		}
	}

	// For objects with types, verify expected type
	if expectedObj != "" {
		if typeStr, ok := result["type"].(string); ok {
			if typeStr != expectedObj {
				log.Error(nil, "Object created with unexpected type",
					"endpoint", endpoint,
					"expected", expectedObj,
					"got", typeStr)
				return nil, fmt.Errorf("object created with unexpected type: %s (expected %s)", typeStr, expectedObj)
			}
		}
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

	// Verify the updated object has an ID field
	if _, ok := result["id"]; !ok {
		log.Error(nil, "Updated object missing ID field",
			"endpoint", endpoint,
			"id", id,
			"keys", getMapKeys(result))

		// As a fallback, retrieve the object we just updated
		log.Info("Fetching updated object as fallback",
			"endpoint", endpoint,
			"id", id)
		return c.GetObject(endpoint, id)
	}

	return result, nil
}

// DeleteObject deletes an object from the AWX API
func (c *Client) DeleteObject(endpoint string, id int) error {
	url := fmt.Sprintf("%s/%d/", endpoint, id)

	// First verify the object exists
	_, err := c.GetObject(endpoint, id)
	if err != nil {
		// If the error indicates the object doesn't exist, treat as success
		if strings.Contains(err.Error(), "404") {
			log.Info("Object already deleted or doesn't exist",
				"endpoint", endpoint,
				"id", id)
			return nil
		}
		return fmt.Errorf("failed to verify object before deletion: %w", err)
	}

	// Object exists, attempt to delete it
	respBody, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		// Check if error is a 404 (already deleted), which can be treated as success
		if strings.Contains(err.Error(), "404") {
			log.Info("Object already deleted",
				"endpoint", endpoint,
				"id", id)
			return nil
		}
		return fmt.Errorf("failed to delete object: %w", err)
	}

	// Per AWX API docs, a successful delete typically returns an empty response
	// But let's add extra handling for any response we might get
	if len(respBody) > 0 {
		log.Info("Delete operation returned non-empty response",
			"endpoint", endpoint,
			"id", id,
			"responseLength", len(respBody))

		// Try to parse the response just in case it contains useful information
		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err == nil {
			if len(result) > 0 {
				log.Info("Delete operation returned structured data",
					"endpoint", endpoint,
					"id", id,
					"keys", getMapKeys(result))
			}
		}
	}

	// Verify the object was actually deleted
	verifyObj, verifyErr := c.GetObject(endpoint, id)
	if verifyErr == nil && verifyObj != nil {
		// Object still exists
		log.Error(nil, "Object still exists after deletion attempt",
			"endpoint", endpoint,
			"id", id)
		return fmt.Errorf("object still exists after deletion attempt")
	}

	log.Info("Successfully deleted object",
		"endpoint", endpoint,
		"id", id)
	return nil
}

// FindObjectByName finds an object by name in the AWX API
func (c *Client) FindObjectByName(endpoint, name string) (map[string]interface{}, error) {
	filters := map[string]string{"name": name}
	objects, err := c.ListObjects(endpoint, filters)
	if err != nil {
		return nil, err
	}

	if len(objects) == 0 {
		// Object not found
		log.Info("Object not found by name",
			"endpoint", endpoint,
			"name", name)
		return nil, nil
	}

	// Per AWX docs, name should be unique, but let's log if we find multiple matches
	if len(objects) > 1 {
		log.Info("Found multiple objects with the same name (using first)",
			"endpoint", endpoint,
			"name", name,
			"count", len(objects))
	}

	// Verify the object has an ID field
	result := objects[0]
	if _, ok := result["id"]; !ok {
		log.Error(nil, "Object returned by API missing ID field",
			"endpoint", endpoint,
			"name", name,
			"keys", getMapKeys(result))

		// Still return the object, but log the issue
		// The calling code should handle objects without IDs
	}

	return result, nil
}

// TestConnection tests the connection to the AWX instance
func (c *Client) TestConnection() error {
	// Make a request to the API v2 endpoint to check if the connection works
	endpoint := "ping"

	log.Info("Testing connection to AWX", "baseURL", c.baseURL)

	// Use the existing doRequest method to leverage our error handling
	respBody, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		log.Error(err, "Failed to connect to AWX",
			"baseURL", c.baseURL,
			"username", c.username)
		return fmt.Errorf("failed to connect to AWX: %w", err)
	}

	// Try to parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err == nil {
		// Check for version or other information
		if version, ok := result["version"]; ok {
			log.Info("Successfully connected to AWX",
				"baseURL", c.baseURL,
				"version", version)
		} else {
			log.Info("Successfully connected to AWX",
				"baseURL", c.baseURL,
				"response", result)
		}
	} else {
		log.Info("Successfully connected to AWX (could not parse response)",
			"baseURL", c.baseURL)
	}

	return nil
}
