package n8napi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:5678/api/v1"
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{},
	}
}

func (c *Client) doReq(method, path string, query url.Values) ([]byte, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("N8N_API_KEY is not set")
	}

	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return nil, err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-N8N-API-KEY", c.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

type WorkflowListItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type WorkflowListResponse struct {
	Data []WorkflowListItem `json:"data"`
}

func (c *Client) ListWorkflows() ([]WorkflowListItem, error) {
	data, err := c.doReq("GET", "/workflows", nil)
	if err != nil {
		return nil, err
	}

	var res WorkflowListResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

type ExecutionListItem struct {
	ID         string  `json:"id"`
	WorkflowID string  `json:"workflowId"`
	Status     string  `json:"status"`
	Mode       string  `json:"mode"`
	StartedAt  string  `json:"startedAt"`
	StoppedAt  *string `json:"stoppedAt"`
}

type ExecutionListResponse struct {
	Data []ExecutionListItem `json:"data"`
}

func (c *Client) ListExecutions(workflowID string, limit string) ([]ExecutionListItem, error) {
	q := url.Values{}
	q.Set("includeData", "false")
	if limit != "" {
		q.Set("limit", limit)
	} else {
		q.Set("limit", "50")
	}
	if workflowID != "" {
		q.Set("workflowId", workflowID)
	}

	data, err := c.doReq("GET", "/executions", q)
	if err != nil {
		return nil, err
	}

	var res ExecutionListResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

func (c *Client) GetExecution(id string) (map[string]any, error) {
	q := url.Values{}
	q.Set("includeData", "true")

	data, err := c.doReq("GET", "/executions/"+id, q)
	if err != nil {
		return nil, err
	}

	var res map[string]any
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) GetWorkflow(id string) (map[string]any, error) {
	data, err := c.doReq("GET", "/workflows/"+id, nil)
	if err != nil {
		return nil, err
	}

	var res map[string]any
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}
