package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type EONETEventParams struct {
	Limit    int    `json:"limit,omitempty"    form:"limit"`
	Days     int    `json:"days,omitempty"     form:"days"`
	Status   string `json:"status,omitempty"   form:"status"`
	Category string `json:"category,omitempty" form:"category"`
}

// --- Response Structs ---

type EONETCategory struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Link        string `json:"link,omitempty"`
}

type EONETSource struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type EONETGeometry struct {
	MagnitudeValue float64     `json:"magnitudeValue,omitempty"`
	MagnitudeUnit  string      `json:"magnitudeUnit,omitempty"`
	Date           string      `json:"date"`
	Type           string      `json:"type"`
	Coordinates    interface{} `json:"coordinates"`
}

type EONETEvent struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Link        string          `json:"link"`
	Closed      string          `json:"closed,omitempty"` // Date when closed
	Categories  []EONETCategory `json:"categories"`
	Sources     []EONETSource   `json:"sources"`
	Geometry    []EONETGeometry `json:"geometry"`
}

type EONETEventResponse struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Link        string       `json:"link"`
	Events      []EONETEvent `json:"events"`
}

type EONETCategoryResponse struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Categories  []EONETCategory `json:"categories"`
}

type EONETClient struct {
	client  *resty.Client
	baseURL string
}

func NewEONETClient() *EONETClient {
	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &EONETClient{
		client:  client,
		baseURL: "https://eonet.gsfc.nasa.gov/api/v3",
	}
}

func (c *EONETClient) FetchEvents(ctx context.Context, params EONETEventParams) (*EONETEventResponse, error) {
	queryParams := make(map[string]string)

	if params.Limit > 0 {
		queryParams["limit"] = strconv.Itoa(params.Limit)
	}
	if params.Days > 0 {
		queryParams["days"] = strconv.Itoa(params.Days)
	}
	if params.Status != "" {
		queryParams["status"] = params.Status
	}
	if params.Category != "" {
		queryParams["category"] = params.Category
	}

	var result EONETEventResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(queryParams).
		SetResult(&result).
		Get(c.baseURL + "/events")
	if err != nil {
		return nil, fmt.Errorf("eonet request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("eonet api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

func (c *EONETClient) FetchCategories(ctx context.Context) (*EONETCategoryResponse, error) {
	var result EONETCategoryResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get(c.baseURL + "/categories")
	if err != nil {
		return nil, fmt.Errorf("eonet request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("eonet api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}
