package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type OSDRFilesParams struct {
	Page     int  `json:"page"      form:"page"`
	Size     int  `json:"size"      form:"size"`
	AllFiles bool `json:"all_files" form:"all_files"`
}

type OSDRSearchParams struct {
	Term   string `json:"term"   form:"term"`
	From   int    `json:"from"   form:"from"`
	Size   int    `json:"size"   form:"size"`
	Type   string `json:"type"   form:"type"`
	Sort   string `json:"sort"   form:"sort"`
	Order  string `json:"order"  form:"order"`
	FField string `json:"ffield" form:"ffield"`
	FValue string `json:"fvalue" form:"fvalue"`
}

type OSDRStudyFile struct {
	Category     string      `json:"category"`
	Subcategory  string      `json:"subcategory"`
	Subdirectory string      `json:"subdirectory"`
	DateCreated  float64     `json:"date_created"`
	DateUpdated  interface{} `json:"date_updated"`
	FileName     string      `json:"file_name"`
	FileSize     int64       `json:"file_size"`
	Organization string      `json:"organization"`
	RemoteURL    string      `json:"remote_url"`
	Restricted   bool        `json:"restricted"`
	Visible      bool        `json:"visible"`
}

type OSDRStudyFilesData struct {
	FileCount  int             `json:"file_count"`
	StudyFiles []OSDRStudyFile `json:"study_files"`
}

type OSDRFilesResponse struct {
	Hits       int                           `json:"hits"`
	Input      string                        `json:"input"`
	PageNumber int                           `json:"page_number"`
	PageSize   int                           `json:"page_size"`
	PageTotal  int                           `json:"page_total"`
	Studies    map[string]OSDRStudyFilesData `json:"studies"`
	Success    bool                          `json:"success"`
	TotalHits  int                           `json:"total_hits"`
}

type OSDRClient struct {
	client  *resty.Client
	baseURL string
}

func NewOSDRClient() *OSDRClient {
	client := resty.New().
		SetTimeout(15 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &OSDRClient{
		client:  client,
		baseURL: "https://osdr.nasa.gov",
	}
}

func (c *OSDRClient) FetchStudyFiles(ctx context.Context, ids string, params OSDRFilesParams) (*OSDRFilesResponse, error) {
	if ids == "" {
		return nil, fmt.Errorf("study ids parameter cannot be empty")
	}

	queryParams := make(map[string]string)
	if params.Page > 0 {
		queryParams["page"] = strconv.Itoa(params.Page)
	}
	if params.Size > 0 {
		queryParams["size"] = strconv.Itoa(params.Size)
	}
	if params.AllFiles {
		queryParams["all_files"] = "true"
	}

	var result OSDRFilesResponse
	endpoint := fmt.Sprintf("%s/osdr/data/osd/files/%s", c.baseURL, ids)

	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(queryParams).
		SetResult(&result).
		Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("osdr files request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("osdr api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

func (c *OSDRClient) FetchStudyMetadata(ctx context.Context, id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("study id parameter cannot be empty")
	}

	var result map[string]interface{}
	endpoint := fmt.Sprintf("%s/osdr/data/osd/meta/%s", c.baseURL, id)

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("osdr metadata request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("osdr api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return result, nil
}

func (c *OSDRClient) SearchDatasets(ctx context.Context, params OSDRSearchParams) (map[string]interface{}, error) {
	queryParams := make(map[string]string)

	if params.Term != "" {
		queryParams["term"] = params.Term
	}
	if params.From > 0 {
		queryParams["from"] = strconv.Itoa(params.From)
	}
	if params.Size > 0 {
		queryParams["size"] = strconv.Itoa(params.Size)
	}
	if params.Type != "" {
		queryParams["type"] = params.Type
	}
	if params.Sort != "" {
		queryParams["sort"] = params.Sort
	}
	if params.Order != "" {
		queryParams["order"] = params.Order
	}
	if params.FField != "" && params.FValue != "" {
		queryParams["ffield"] = params.FField
		queryParams["fvalue"] = params.FValue
	}

	var result map[string]interface{}
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(queryParams).
		SetResult(&result).
		Get(c.baseURL + "/osdr/data/search")
	if err != nil {
		return nil, fmt.Errorf("osdr search request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("osdr search api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return result, nil
}

func (c *OSDRClient) FetchEntityAll(ctx context.Context, entityType string) (interface{}, error) {
	var result interface{}
	endpoint := fmt.Sprintf("%s/geode-py/ws/api/%s", c.baseURL, entityType)

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("geode-py entity request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("geode-py api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return result, nil
}

func (c *OSDRClient) FetchEntitySingle(ctx context.Context, entityTypeSingular, id string) (interface{}, error) {
	var result interface{}
	endpoint := fmt.Sprintf("%s/geode-py/ws/api/%s/%s", c.baseURL, entityTypeSingular, id)

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("geode-py single entity request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("geode-py api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return result, nil
}
