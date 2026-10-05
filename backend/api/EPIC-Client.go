package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type CentroidCoordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Position3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type AttitudeQuaternions struct {
	Q0 float64 `json:"q0"`
	Q1 float64 `json:"q1"`
	Q2 float64 `json:"q2"`
	Q3 float64 `json:"q3"`
}

type EPICImage struct {
	Image               string              `json:"image"`
	Caption             string              `json:"caption"`
	Date                string              `json:"date"`
	CentroidCoordinates CentroidCoordinates `json:"centroid_coordinates"`
	DscovrJ2000Position Position3D          `json:"dscovr_j2000_position"`
	LunarJ2000Position  Position3D          `json:"lunar_j2000_position"`
	SunJ2000Position    Position3D          `json:"sun_j2000_position"`
	AttitudeQuaternions AttitudeQuaternions `json:"attitude_quaternions"`
	ImageURL            string              `json:"image_url,omitempty"`
}

type EPICDate struct {
	Date string `json:"date"`
}

type EPICClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

func NewEPICClient(apiKey string) *EPICClient {
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}

	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &EPICClient{
		client:  client,
		baseURL: "https://api.nasa.gov/EPIC",
		apiKey:  apiKey,
	}
}

func (c *EPICClient) BuildImageURL(imageType, dateStr, imageName string) string {
	parts := strings.Split(dateStr, " ")
	if len(parts) > 0 {
		dateParts := strings.Split(parts[0], "-")
		if len(dateParts) == 3 {
			return fmt.Sprintf("https://api.nasa.gov/EPIC/archive/%s/%s/%s/%s/png/%s.png?api_key=%s",
				imageType, dateParts[0], dateParts[1], dateParts[2], imageName, c.apiKey)
		}
	}
	return ""
}

// FetchRecent retrieves the most recent imagery metadata for "natural" or "enhanced" color.
func (c *EPICClient) FetchRecent(ctx context.Context, imageType string) ([]EPICImage, error) {
	if imageType == "" {
		imageType = "natural"
	}

	var results []EPICImage
	endpoint := fmt.Sprintf("%s/api/%s", c.baseURL, imageType)

	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParam("api_key", c.apiKey).
		SetResult(&results).
		Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("epic request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("epic api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	for i := range results {
		results[i].ImageURL = c.BuildImageURL(imageType, results[i].Date, results[i].Image)
	}

	return results, nil
}

func (c *EPICClient) FetchByDate(ctx context.Context, imageType, date string) ([]EPICImage, error) {
	if imageType == "" {
		imageType = "natural"
	}

	var results []EPICImage
	endpoint := fmt.Sprintf("%s/api/%s/date/%s", c.baseURL, imageType, date)

	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParam("api_key", c.apiKey).
		SetResult(&results).
		Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("epic request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("epic api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	for i := range results {
		results[i].ImageURL = c.BuildImageURL(imageType, results[i].Date, results[i].Image)
	}

	return results, nil
}

func (c *EPICClient) FetchAvailableDates(ctx context.Context, imageType string) ([]EPICDate, error) {
	if imageType == "" {
		imageType = "natural"
	}

	var results []EPICDate
	endpoint := fmt.Sprintf("%s/api/%s/all", c.baseURL, imageType)

	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParam("api_key", c.apiKey).
		SetResult(&results).
		Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("epic request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("epic api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return results, nil
}
