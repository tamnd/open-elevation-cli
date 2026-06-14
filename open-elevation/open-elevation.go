// Package openelevation is the library behind the elevation command line:
// the HTTP client, request shaping, and the typed data models for the Open
// Elevation API (api.open-elevation.com).
//
// No API key required. The API is public and free.
// The Client sets a real User-Agent, paces requests to stay polite, and
// retries transient failures (429 and 5xx).
package openelevation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "api.open-elevation.com"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.open-elevation.com/api/v1",
		UserAgent: "elevation-cli/0.1.0 (github.com/tamnd/open-elevation-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to the Open Elevation API over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Lookup fetches the elevation for a single GPS coordinate.
func (c *Client) Lookup(ctx context.Context, lat, lon float64) (*Point, error) {
	u := fmt.Sprintf("%s/lookup?locations=%f,%f", c.cfg.BaseURL, lat, lon)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp wireResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode lookup: %w", err)
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results for %.6f,%.6f", lat, lon)
	}
	p := resp.Results[0]
	return &p, nil
}

// Batch fetches elevations for multiple GPS coordinates in one POST request.
// coords is a slice of [lat, lon] pairs.
func (c *Client) Batch(ctx context.Context, coords [][2]float64) ([]Point, error) {
	var body wirePostBody
	for _, c := range coords {
		body.Locations = append(body.Locations, struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: c[0], Longitude: c[1]})
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode batch: %w", err)
	}
	u := c.cfg.BaseURL + "/lookup"
	respBody, err := c.post(ctx, u, data)
	if err != nil {
		return nil, err
	}
	var resp wireResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("decode batch: %w", err)
	}
	return resp.Results, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) post(ctx context.Context, url string, body []byte) ([]byte, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// Point is one elevation result from the Open Elevation API.
type Point struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Elevation float64 `json:"elevation"`
}

// wirePostBody is the JSON body for a batch POST request.
type wirePostBody struct {
	Locations []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"locations"`
}

// wireResp is the JSON response from the API (both GET and POST).
type wireResp struct {
	Results []Point `json:"results"`
}
