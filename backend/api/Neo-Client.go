package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type NeoWsClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

func NewNeoWsClient(apiKey string) *NeoWsClient {
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}

	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &NeoWsClient{
		client:  client,
		baseURL: "https://api.nasa.gov/neo/rest/v1",
		apiKey:  apiKey,
	}
}

func (c *NeoWsClient) FetchFeed(ctx context.Context, params NeoFeedParams) (*NeoFeedResponse, error) {
	var result NeoFeedResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(params.ToQueryMap()).
		SetResult(&result).
		Get(c.baseURL + "/feed")
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("nasa neo api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

func (c *NeoWsClient) FetchAsteroidByID(ctx context.Context, asteroidID string, apiKey string) (*NearEarthObject, error) {
	if asteroidID == "" {
		return nil, errors.New("asteroid_id cannot be empty")
	}

	if apiKey == "" {
		apiKey = c.apiKey
	}

	var result NearEarthObject

	url := fmt.Sprintf("%s/neo/%s", c.baseURL, asteroidID)
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParam("api_key", apiKey).
		SetResult(&result).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("nasa neo api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}
