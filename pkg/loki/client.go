package loki

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	baseURL string
	client  *resty.Client
}

type QueryResponse struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Stream `json:"result"`
}

type Stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  resty.New().SetTimeout(30 * time.Second),
	}
}

// QueryRange queries Loki for logs in a time range
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, limit int) ([]string, error) {
	var queryResp QueryResponse

	req := c.client.R().
		SetContext(ctx).
		SetQueryParam("query", query).
		SetQueryParam("start", fmt.Sprintf("%d", start.UnixNano())).
		SetQueryParam("end", fmt.Sprintf("%d", end.UnixNano())).
		SetResult(&queryResp)

	if limit > 0 {
		req.SetQueryParam("limit", fmt.Sprintf("%d", limit))
	}

	resp, err := req.Get(c.baseURL + "/loki/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode(), resp.String())
	}

	var logs []string
	for _, stream := range queryResp.Data.Result {
		for _, value := range stream.Values {
			if len(value) >= 2 {
				logs = append(logs, value[1])
			}
		}
	}

	return logs, nil
}
