package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type APODResponse struct {
	Copyright      string `json:"copyright,omitempty"`
	Date           string `json:"date"`
	Explanation    string `json:"explanation"`
	HDURL          string `json:"hdurl,omitempty"`
	MediaType      string `json:"media_type"`
	ServiceVersion string `json:"service_version"`
	Title          string `json:"title"`
	URL            string `json:"url"`
}

type APODClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

func NewAPODClient(apiKey string) *APODClient {
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}

	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &APODClient{
		client:  client,
		baseURL: "https://api.nasa.gov/planetary/apod",
		apiKey:  apiKey,
	}
}

func (c *APODClient) FetchAPOD(ctx context.Context, params APODParams) ([]APODResponse, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	queryMap := params.ToQueryMap()
	if _, exists := queryMap["api_key"]; !exists {
		queryMap["api_key"] = c.apiKey
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(queryMap).
		Get(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("nasa api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	body := resp.Body()

	isMulti := params.Count > 0 || params.StartDate != ""

	if isMulti {
		var results []APODResponse
		if err := json.Unmarshal(body, &results); err != nil {
			return nil, fmt.Errorf("failed to parse array response: %w", err)
		}
		return results, nil
	}

	var single APODResponse
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, fmt.Errorf("failed to parse single response: %w", err)
	}

	return []APODResponse{single}, nil
}
