package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type BaseDONKIParams struct {
	StartDate string `json:"startDate,omitempty" form:"startDate"`
	EndDate   string `json:"endDate,omitempty"   form:"endDate"`
	APIKey    string `json:"api_key,omitempty"   form:"api_key"`
}

type CMEAnalysisParams struct {
	BaseDONKIParams
	MostAccurateOnly  bool   `json:"mostAccurateOnly" form:"mostAccurateOnly"`
	CompleteEntryOnly bool   `json:"completeEntryOnly" form:"completeEntryOnly"`
	Speed             int    `json:"speed"            form:"speed"`
	HalfAngle         int    `json:"halfAngle"        form:"halfAngle"`
	Catalog           string `json:"catalog"          form:"catalog"`
	Keyword           string `json:"keyword"          form:"keyword"`
}

type IPSParams struct {
	BaseDONKIParams
	Location string `json:"location" form:"location"`
	Catalog  string `json:"catalog"  form:"catalog"`
}

type NotificationParams struct {
	BaseDONKIParams
	Type string `json:"type" form:"type"`
}

type Instrument struct {
	DisplayName string `json:"displayName"`
}

type CoronalMassEjection struct {
	ActivityID      string       `json:"activityID"`
	Catalog         string       `json:"catalog"`
	StartTime       string       `json:"startTime"`
	SourceLocation  string       `json:"sourceLocation"`
	ActiveRegionNum int          `json:"activeRegionNum"`
	Link            string       `json:"link"`
	Note            string       `json:"note"`
	Instruments     []Instrument `json:"instruments"`
}

type SolarFlare struct {
	FlareID         string       `json:"flrID"`
	Instruments     []Instrument `json:"instruments"`
	BeginTime       string       `json:"beginTime"`
	PeakTime        string       `json:"peakTime"`
	EndTime         string       `json:"endTime"`
	ClassType       string       `json:"classType"`
	SourceLocation  string       `json:"sourceLocation"`
	ActiveRegionNum int          `json:"activeRegionNum"`
	Link            string       `json:"link"`
}

type GeomagneticStorm struct {
	GSTID     string `json:"gstID"`
	StartTime string `json:"startTime"`
	Link      string `json:"link"`
}

type DONKINotification struct {
	NotificationID   string `json:"messageID"`
	NotificationType string `json:"messageType"`
	IssueTime        string `json:"messageIssueTime"`
	URL              string `json:"messageURL"`
	Body             string `json:"messageBody"`
}

type DONKIClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

func NewDONKIClient(apiKey string) *DONKIClient {
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}

	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &DONKIClient{
		client:  client,
		baseURL: "https://api.nasa.gov/DONKI",
		apiKey:  apiKey,
	}
}

func (c *DONKIClient) buildQueryParams(base BaseDONKIParams) map[string]string {
	params := make(map[string]string)
	apiKey := base.APIKey
	if apiKey == "" {
		apiKey = c.apiKey
	}
	params["api_key"] = apiKey

	if base.StartDate != "" {
		params["startDate"] = base.StartDate
	}
	if base.EndDate != "" {
		params["endDate"] = base.EndDate
	}
	return params
}

func (c *DONKIClient) FetchCME(ctx context.Context, params BaseDONKIParams) ([]CoronalMassEjection, error) {
	var results []CoronalMassEjection
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(c.buildQueryParams(params)).
		SetResult(&results).
		Get(c.baseURL + "/CME")
	if err != nil {
		return nil, fmt.Errorf("donki request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("donki api error: status %d - %s", resp.StatusCode(), resp.String())
	}
	return results, nil
}

func (c *DONKIClient) FetchSolarFlares(ctx context.Context, params BaseDONKIParams) ([]SolarFlare, error) {
	var results []SolarFlare
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(c.buildQueryParams(params)).
		SetResult(&results).
		Get(c.baseURL + "/FLR")
	if err != nil {
		return nil, fmt.Errorf("donki request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("donki api error: status %d - %s", resp.StatusCode(), resp.String())
	}
	return results, nil
}

func (c *DONKIClient) FetchGeomagneticStorms(ctx context.Context, params BaseDONKIParams) ([]GeomagneticStorm, error) {
	var results []GeomagneticStorm
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(c.buildQueryParams(params)).
		SetResult(&results).
		Get(c.baseURL + "/GST")
	if err != nil {
		return nil, fmt.Errorf("donki request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("donki api error: status %d - %s", resp.StatusCode(), resp.String())
	}
	return results, nil
}

func (c *DONKIClient) FetchNotifications(ctx context.Context, params NotificationParams) ([]DONKINotification, error) {
	queryParams := c.buildQueryParams(params.BaseDONKIParams)
	if params.Type != "" {
		queryParams["type"] = params.Type
	}

	var results []DONKINotification
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(queryParams).
		SetResult(&results).
		Get(c.baseURL + "/notifications")
	if err != nil {
		return nil, fmt.Errorf("donki request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("donki api error: status %d - %s", resp.StatusCode(), resp.String())
	}
	return results, nil
}
