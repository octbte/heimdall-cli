package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) get(path string, params url.Values, out any) error {
	u, err := url.Parse(c.baseURL + "/api/v1" + path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if len(params) > 0 {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.NewDecoder(resp.Body).Decode(&errResp); jsonErr == nil && errResp.Error.Message != "" {
			return fmt.Errorf("API error %d (%s): %s", resp.StatusCode, errResp.Error.Code, errResp.Error.Message)
		}
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func setIfNotEmpty(params url.Values, key, val string) {
	if val != "" {
		params.Set(key, val)
	}
}

func (c *Client) GetOverview(from, to string) (*OverviewResponse, error) {
	params := url.Values{}
	setIfNotEmpty(params, "from", from)
	setIfNotEmpty(params, "to", to)
	var out OverviewResponse
	return &out, c.get("/dashboard/overview", params, &out)
}

func (c *Client) ListEvents(p ListEventsParams) (*EventsResponse, error) {
	params := url.Values{}
	setIfNotEmpty(params, "level", p.Level)
	setIfNotEmpty(params, "status", p.Status)
	setIfNotEmpty(params, "event_name", p.EventName)
	setIfNotEmpty(params, "environment_id", p.Environment)
	setIfNotEmpty(params, "from", p.From)
	setIfNotEmpty(params, "to", p.To)
	if p.Limit > 0 {
		params.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset > 0 {
		params.Set("offset", strconv.Itoa(p.Offset))
	}
	var out EventsResponse
	return &out, c.get("/events", params, &out)
}

func (c *Client) GetEvent(id string) (*EventItem, error) {
	var out EventItem
	return &out, c.get("/events/"+id, nil, &out)
}

func (c *Client) ListErrors(p ListErrorsParams) (*ErrorsResponse, error) {
	params := url.Values{}
	setIfNotEmpty(params, "level", p.Level)
	setIfNotEmpty(params, "fingerprint", p.Fingerprint)
	setIfNotEmpty(params, "endpoint", p.Endpoint)
	setIfNotEmpty(params, "job_name", p.JobName)
	setIfNotEmpty(params, "environment_id", p.Environment)
	setIfNotEmpty(params, "from", p.From)
	setIfNotEmpty(params, "to", p.To)
	if p.Limit > 0 {
		params.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset > 0 {
		params.Set("offset", strconv.Itoa(p.Offset))
	}
	var out ErrorsResponse
	return &out, c.get("/errors", params, &out)
}

func (c *Client) GetError(id string) (*ErrorItem, error) {
	var out ErrorItem
	return &out, c.get("/errors/"+id, nil, &out)
}

func (c *Client) ListPerformance(p ListPerfParams) (*PerfResponse, error) {
	params := url.Values{}
	setIfNotEmpty(params, "metric_name", p.MetricName)
	setIfNotEmpty(params, "target_type", p.TargetType)
	setIfNotEmpty(params, "target_name", p.TargetName)
	setIfNotEmpty(params, "environment_id", p.Environment)
	setIfNotEmpty(params, "from", p.From)
	setIfNotEmpty(params, "to", p.To)
	if p.Limit > 0 {
		params.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset > 0 {
		params.Set("offset", strconv.Itoa(p.Offset))
	}
	var out PerfResponse
	return &out, c.get("/performance", params, &out)
}

func (c *Client) GetPerformance(id string) (*PerfItem, error) {
	var out PerfItem
	return &out, c.get("/performance/"+id, nil, &out)
}

func (c *Client) GetTrace(traceID string) (*TraceResponse, error) {
	var out TraceResponse
	return &out, c.get("/traces/"+traceID, nil, &out)
}
