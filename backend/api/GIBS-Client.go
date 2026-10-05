package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type GIBSTileParams struct {
	Layer         string `json:"layer"         form:"layer"`
	Projection    string `json:"projection"    form:"projection"`
	Date          string `json:"date"          form:"date"`
	TileMatrixSet string `json:"tileMatrixSet" form:"tileMatrixSet"`
	Zoom          int    `json:"z"             form:"z"`
	Row           int    `json:"r"             form:"r"`
	Col           int    `json:"c"             form:"c"`
	Format        string `json:"format"        form:"format"`
}

type GIBSTileURLResponse struct {
	Layer      string `json:"layer"`
	Projection string `json:"projection"`
	Date       string `json:"date"`
	TileURL    string `json:"tile_url"`
}

type GIBSClient struct {
	client  *resty.Client
	baseURL string
}

func NewGIBSClient() *GIBSClient {
	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second)

	return &GIBSClient{
		client:  client,
		baseURL: "https://gibs.earthdata.nasa.gov",
	}
}

func (c *GIBSClient) BuildTileURL(params GIBSTileParams) GIBSTileURLResponse {
	proj := params.Projection
	if proj == "" {
		proj = "epsg4326"
	}

	layer := params.Layer
	if layer == "" {
		layer = "MODIS_Terra_CorrectedReflectance_TrueColor"
	}

	date := params.Date
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	tileMatrixSet := params.TileMatrixSet
	if tileMatrixSet == "" {
		tileMatrixSet = "250m"
	}

	format := params.Format
	if format == "" {
		format = "jpg"
	}

	tileURL := fmt.Sprintf("%s/wmts/%s/best/%s/default/%s/%s/%d/%d/%d.%s",
		c.baseURL, proj, layer, date, tileMatrixSet, params.Zoom, params.Row, params.Col, format)

	return GIBSTileURLResponse{
		Layer:      layer,
		Projection: proj,
		Date:       date,
		TileURL:    tileURL,
	}
}

func (c *GIBSClient) FetchCapabilities(ctx context.Context, projection string) (string, error) {
	if projection == "" {
		projection = "epsg4326"
	}

	url := fmt.Sprintf("%s/wmts/%s/best/1.0.0/WMTSCapabilities.xml", c.baseURL, projection)

	resp, err := c.client.R().
		SetContext(ctx).
		Get(url)
	if err != nil {
		return "", fmt.Errorf("gibs capabilities request failed: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("gibs api error: status %d - %s", resp.StatusCode(), resp.String())
	}

	return resp.String(), nil
}
