package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// One month of regional temperature. Pointers are nil when NASA has no value.
type TempPoint struct {
	Period string   `json:"p"` // YYYYMM
	T2M    *float64 `json:"t2m"`
	TS     *float64 `json:"ts"`
}

type RegionalTemperature struct {
	Source string      `json:"source"`
	Points int         `json:"points"`
	HasTS  bool        `json:"hasTS"`
	BBox   [4]float64  `json:"bbox"` // south, north, west, east
	Series []TempPoint `json:"series"`
}

type climateCacheEntry struct {
	data    *RegionalTemperature
	expires time.Time
}

type ClimateClient struct {
	client  *resty.Client
	baseURL string
	mu      sync.Mutex
	cache   map[string]climateCacheEntry
}

func NewClimateClient() *ClimateClient {
	return &ClimateClient{
		client:  resty.New().SetTimeout(40 * time.Second),
		baseURL: "https://power.larc.nasa.gov/api/temporal/monthly/point",
		cache:   make(map[string]climateCacheEntry),
	}
}

type powerResponse struct {
	Properties struct {
		Parameter map[string]map[string]float64 `json:"parameter"`
	} `json:"properties"`
}

// fetchPoint returns parameter -> YYYYMM -> value for one location.
func (c *ClimateClient) fetchPoint(ctx context.Context, lat, lng float64, params string) (map[string]map[string]float64, error) {
	var out powerResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"parameters": params,
			"community":  "RE",
			"longitude":  fmt.Sprintf("%.4f", lng),
			"latitude":   fmt.Sprintf("%.4f", lat),
			"start":      "1981",
			"end":        fmt.Sprintf("%d", time.Now().UTC().Year()),
			"format":     "JSON",
		}).
		SetResult(&out).
		Get(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("power request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("power api error: status %d", resp.StatusCode())
	}
	return out.Properties.Parameter, nil
}

// FetchRegionalTemperature averages NASA POWER monthly values over a grid of
// sample points inside the bounding box (3x3 for larger views, 1 point for tiny ones).
func (c *ClimateClient) FetchRegionalTemperature(ctx context.Context, south, north, west, east float64) (*RegionalTemperature, error) {
	if south >= north || west >= east {
		return nil, errors.New("invalid bounding box")
	}
	if south < -90 || north > 90 {
		return nil, errors.New("latitude must be between -90 and 90")
	}
	if north-south > 30 || east-west > 30 {
		return nil, errors.New("region too large: zoom in so the view is under 30 degrees wide")
	}

	key := fmt.Sprintf("%.1f|%.1f|%.1f|%.1f", south, north, west, east)
	c.mu.Lock()
	if e, ok := c.cache[key]; ok && time.Now().Before(e.expires) {
		c.mu.Unlock()
		return e.data, nil
	}
	c.mu.Unlock()

	// Build sample points.
	type pt struct{ lat, lng float64 }
	var pts []pt
	if north-south < 2 && east-west < 2 {
		pts = []pt{{(south + north) / 2, (west + east) / 2}}
	} else {
		for _, fy := range []float64{1.0 / 6, 0.5, 5.0 / 6} {
			for _, fx := range []float64{1.0 / 6, 0.5, 5.0 / 6} {
				pts = append(pts, pt{south + (north-south)*fy, west + (east-west)*fx})
			}
		}
	}
	for i := range pts {
		pts[i].lng = math.Mod(math.Mod(pts[i].lng+540, 360)+360, 360) - 180 // wrap to -180..180
	}

	type result struct {
		params map[string]map[string]float64
		err    error
	}
	results := make([]result, len(pts))
	var wg sync.WaitGroup
	for i, p := range pts {
		wg.Add(1)
		go func(i int, p pt) {
			defer wg.Done()
			params, err := c.fetchPoint(ctx, p.lat, p.lng, "T2M,TS")
			if err != nil {
				// If TS is not accepted, retry with air temperature only.
				params, err = c.fetchPoint(ctx, p.lat, p.lng, "T2M")
			}
			results[i] = result{params, err}
		}(i, p)
	}
	wg.Wait()

	type acc struct{ sum, n float64 }
	t2m := map[string]*acc{}
	ts := map[string]*acc{}
	okPoints := 0
	var lastErr error
	add := func(m map[string]*acc, series map[string]float64) {
		for period, v := range series {
			if len(period) != 6 || strings.HasSuffix(period, "13") || v <= -900 || math.IsNaN(v) {
				continue // skip annual rows (YYYY13) and missing values (-999)
			}
			a := m[period]
			if a == nil {
				a = &acc{}
				m[period] = a
			}
			a.sum += v
			a.n++
		}
	}
	for _, r := range results {
		if r.err != nil {
			lastErr = r.err
			continue
		}
		okPoints++
		add(t2m, r.params["T2M"])
		add(ts, r.params["TS"])
	}
	if okPoints == 0 {
		if lastErr == nil {
			lastErr = errors.New("no data returned")
		}
		return nil, lastErr
	}

	periods := make([]string, 0, len(t2m))
	seen := map[string]bool{}
	for p := range t2m {
		periods = append(periods, p)
		seen[p] = true
	}
	for p := range ts {
		if !seen[p] {
			periods = append(periods, p)
		}
	}
	sort.Strings(periods)

	round := func(v float64) *float64 { r := math.Round(v*100) / 100; return &r }
	series := make([]TempPoint, 0, len(periods))
	for _, p := range periods {
		tp := TempPoint{Period: p}
		if a := t2m[p]; a != nil && a.n > 0 {
			tp.T2M = round(a.sum / a.n)
		}
		if a := ts[p]; a != nil && a.n > 0 {
			tp.TS = round(a.sum / a.n)
		}
		series = append(series, tp)
	}

	data := &RegionalTemperature{
		Source: "NASA POWER (MERRA-2 reanalysis model), monthly",
		Points: okPoints,
		HasTS:  len(ts) > 0,
		BBox:   [4]float64{south, north, west, east},
		Series: series,
	}

	c.mu.Lock()
	c.cache[key] = climateCacheEntry{data: data, expires: time.Now().Add(6 * time.Hour)}
	c.mu.Unlock()
	return data, nil
}
