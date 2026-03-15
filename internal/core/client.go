package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/clash-proxyd/proxyd/internal/types"
)

// Client interacts with mihomo RESTful API
type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

// NewClient creates a new mihomo API client
func NewClient(baseURL, secret string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetVersion returns mihomo version
func (c *Client) GetVersion() (string, error) {
	var result map[string]interface{}
	if err := c.get("/version", &result); err != nil {
		return "", err
	}

	if version, ok := result["version"].(string); ok {
		return version, nil
	}

	return "", fmt.Errorf("version not found in response")
}

// GetTraffic returns traffic statistics
func (c *Client) GetTraffic() (*types.MihomoTraffic, error) {
	var traffic types.MihomoTraffic
	if err := c.get("/traffic", &traffic); err != nil {
		return nil, err
	}
	return &traffic, nil
}

// GetMemory returns memory usage
func (c *Client) GetMemory() (*types.MihomoMemory, error) {
	var memory types.MihomoMemory
	if err := c.get("/memory", &memory); err != nil {
		return nil, err
	}
	return &memory, nil
}

// GetProxies returns all proxies
func (c *Client) GetProxies() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.get("/proxies", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetProxy returns a specific proxy
func (c *Client) GetProxy(name string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.get("/proxies/"+name, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetProxyDelay returns proxy delay
func (c *Client) GetProxyDelay(proxyName string, testURL string, timeout int) (int, error) {
	url := fmt.Sprintf("/proxies/%s/delay", proxyName)

	body := map[string]interface{}{
		"url":     testURL,
		"timeout": timeout,
	}

	var result map[string]interface{}
	if err := c.post(url, body, &result); err != nil {
		return 0, err
	}

	if delay, ok := result["delay"].(float64); ok {
		return int(delay), nil
	}

	return 0, fmt.Errorf("delay not found in response")
}

// SwitchProxy switches to a proxy in a group
func (c *Client) SwitchProxy(group, proxy string) error {
	url := fmt.Sprintf("/proxies/%s", group)

	body := map[string]string{
		"name": proxy,
	}

	return c.put(url, body, nil)
}

// GetConnections returns all active connections
func (c *Client) GetConnections() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.get("/connections", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CloseConnection closes a specific connection by ID
func (c *Client) CloseConnection(id string) error {
	return c.doRequest("DELETE", "/connections/"+id, nil, nil)
}

// CloseAllConnections closes all active connections
func (c *Client) CloseAllConnections() error {
	return c.doRequest("DELETE", "/connections", nil, nil)
}

// GetRules returns all rules
func (c *Client) GetRules() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.get("/rules", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// getConfig returns mihomo configuration
func (c *Client) getConfig() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.get("/configs", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// updateConfig updates mihomo configuration
func (c *Client) updateConfig(config map[string]interface{}) error {
	return c.patch("/configs", config, nil)
}

// get performs a GET request
func (c *Client) get(path string, result interface{}) error {
	return c.doRequest("GET", path, nil, result)
}

// post performs a POST request
func (c *Client) post(path string, body, result interface{}) error {
	return c.doRequest("POST", path, body, result)
}

// put performs a PUT request
func (c *Client) put(path string, body, result interface{}) error {
	return c.doRequest("PUT", path, body, result)
}

// patch performs a PATCH request
func (c *Client) patch(path string, body, result interface{}) error {
	return c.doRequest("PATCH", path, body, result)
}

// doRequest performs an HTTP request
func (c *Client) doRequest(method, path string, body, result interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
