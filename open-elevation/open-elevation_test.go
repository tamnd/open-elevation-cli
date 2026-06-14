package openelevation_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	openelevation "github.com/tamnd/open-elevation-cli/open-elevation"
)

const fakeLookupJSON = `{"results":[{"latitude":41.161758,"longitude":-8.583933,"elevation":117}]}`

const fakeBatchJSON = `{"results":[
  {"latitude":41.161758,"longitude":-8.583933,"elevation":117},
  {"latitude":10,"longitude":10,"elevation":65},
  {"latitude":-20,"longitude":30,"elevation":832}
]}`

func newTestClient(ts *httptest.Server) *openelevation.Client {
	cfg := openelevation.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return openelevation.NewClient(cfg)
}

func TestLookupSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeLookupJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Lookup(context.Background(), 41.161758, -8.583933)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestLookupParsesPoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeLookupJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	pt, err := c.Lookup(context.Background(), 41.161758, -8.583933)
	if err != nil {
		t.Fatal(err)
	}
	if pt.Latitude != 41.161758 {
		t.Errorf("Latitude = %f, want 41.161758", pt.Latitude)
	}
	if pt.Longitude != -8.583933 {
		t.Errorf("Longitude = %f, want -8.583933", pt.Longitude)
	}
	if pt.Elevation != 117 {
		t.Errorf("Elevation = %f, want 117", pt.Elevation)
	}
}

func TestBatchParsesMultiple(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		_, _ = fmt.Fprint(w, fakeBatchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	pts, err := c.Batch(context.Background(), [][2]float64{
		{41.161758, -8.583933},
		{10, 10},
		{-20, 30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 3 {
		t.Fatalf("len(pts) = %d, want 3", len(pts))
	}
	if pts[0].Elevation != 117 {
		t.Errorf("pts[0].Elevation = %f, want 117", pts[0].Elevation)
	}
	if pts[2].Elevation != 832 {
		t.Errorf("pts[2].Elevation = %f, want 832", pts[2].Elevation)
	}
}

func TestLookupRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeLookupJSON)
	}))
	defer ts.Close()

	cfg := openelevation.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := openelevation.NewClient(cfg)

	_, err := c.Lookup(context.Background(), 41.161758, -8.583933)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestBatchSendsJSON(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		var err error
		gotBody = make([]byte, r.ContentLength)
		_, err = r.Body.Read(gotBody)
		_ = err
		_, _ = fmt.Fprint(w, fakeBatchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Batch(context.Background(), [][2]float64{{41.161758, -8.583933}})
	if err != nil {
		t.Fatal(err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	// body must be valid JSON with a locations array
	var v map[string]any
	if err := json.Unmarshal(gotBody, &v); err != nil {
		t.Errorf("POST body is not valid JSON: %v (body=%q)", err, gotBody)
	}
	if _, ok := v["locations"]; !ok {
		t.Errorf("POST body missing 'locations' key: %s", gotBody)
	}
}
