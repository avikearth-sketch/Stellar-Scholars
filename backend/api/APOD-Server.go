package api

import (
	"errors"
	"strconv"
)

type APODParams struct {
	Date      string `json:"date,omitempty" url:"date,omitempty" form:"date"`
	StartDate string `json:"start_date,omitempty" url:"start_date,omitempty" form:"start_date"`
	EndDate   string `json:"end_date,omitempty" url:"end_date,omitempty" form:"end_date"`
	Count     int    `json:"count,omitempty" url:"count,omitempty" form:"count"`
	APIKey    string `json:"api_key,omitempty" url:"api_key,omitempty" form:"api_key"`
}

func (p *APODParams) Validate() error {
	if p.Count > 25 {
		return errors.New("count parameter cannot exceed 25")
	}

	hasDate := p.Date != ""
	hasRange := p.StartDate != "" || p.EndDate != ""
	hasCount := p.Count > 0

	if hasCount && (hasDate || hasRange) {
		return errors.New("count cannot be used with date, start_date, or end_date")
	}

	if hasDate && hasRange {
		return errors.New("date cannot be used with start_date or end_date")
	}

	return nil
}

func (p *APODParams) ToQueryMap() map[string]string {
	params := make(map[string]string)

	apiKey := p.APIKey
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}
	params["api_key"] = apiKey

	if p.Date != "" {
		params["date"] = p.Date
	}
	if p.StartDate != "" {
		params["start_date"] = p.StartDate
	}
	if p.EndDate != "" {
		params["end_date"] = p.EndDate
	}
	if p.Count > 0 {
		params["count"] = strconv.Itoa(p.Count)
	}

	return params
}
