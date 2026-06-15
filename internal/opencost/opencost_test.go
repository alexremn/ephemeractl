package opencost

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAllocationTotalSumsAllComponentsIncludingPVs(t *testing.T) {
	a := Allocation{
		CPUCost:          1.00,
		RAMCost:          0.50,
		GPUCost:          0.00,
		NetworkCost:      0.10,
		LoadBalancerCost: 0.25,
		SharedCost:       0.05,
		PVs: map[string]pvCost{
			"pv-a": {Cost: 0.30},
			"pv-b": {Cost: 0.20},
		},
	}
	got := a.Total()
	want := 1.00 + 0.50 + 0.10 + 0.25 + 0.05 + 0.30 + 0.20 // = 2.40
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Total() = %v, want %v", got, want)
	}
}

func TestFetchBuildsQueryAndReducesResult(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/allocation" {
			t.Errorf("path = %q, want /allocation", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "code": 200,
		  "data": [
		    {
		      "frontend": {"name":"frontend","cpuCost":1.0,"ramCost":0.5,"pvs":{"d":{"cost":0.25}}},
		      "backend":  {"name":"backend","cpuCost":2.0,"ramCost":0.0,"networkCost":0.1}
		    }
		  ]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	res, err := c.Fetch(context.Background(), Query{
		Window: "today", Resolution: "1m",
		LabelKey: "ephemeractl.dev/pr", LabelValue: "482",
		TeamLabel: "team",
	})
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	// Query construction
	if gotQuery.Get("window") != "today" || gotQuery.Get("accumulate") != "true" ||
		gotQuery.Get("resolution") != "1m" {
		t.Errorf("base query wrong: %v", gotQuery)
	}
	if gotQuery.Get("filterLabels") != "ephemeractl.dev/pr:482" {
		t.Errorf("filterLabels = %q", gotQuery.Get("filterLabels"))
	}
	if gotQuery.Get("aggregate") != "label:team" {
		t.Errorf("aggregate = %q", gotQuery.Get("aggregate"))
	}

	// Reduction: 1.0+0.5+0.25 (frontend) + 2.0+0.1 (backend) = 3.85
	if math.Abs(res.Total-3.85) > 1e-9 {
		t.Errorf("Total = %v, want 3.85", res.Total)
	}
	if len(res.Groups) != 2 {
		t.Fatalf("len(Groups) = %d, want 2", len(res.Groups))
	}
	if res.Groups[0].Name != "backend" { // sorted desc by cost: backend 2.1 > frontend 1.75
		t.Errorf("Groups[0] = %q, want backend (highest cost first)", res.Groups[0].Name)
	}
	if res.Components.CPU < 2.99 || res.Components.CPU > 3.01 { // 1.0 + 2.0
		t.Errorf("Components.CPU = %v, want 3.0", res.Components.CPU)
	}
	if res.Components.PV < 0.24 || res.Components.PV > 0.26 {
		t.Errorf("Components.PV = %v, want 0.25", res.Components.PV)
	}
}

func TestFetchNamespaceSelectorAndIdle(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"code":200,"data":[{}]}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL).Fetch(context.Background(), Query{
		Window: "7d", Resolution: "1m", Namespace: "preview-pr-7", IncludeIdle: true,
	})
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if gotQuery.Get("filterNamespaces") != "preview-pr-7" {
		t.Errorf("filterNamespaces = %q", gotQuery.Get("filterNamespaces"))
	}
	if gotQuery.Get("aggregate") != "namespace" { // no team-label → aggregate by namespace
		t.Errorf("aggregate = %q, want namespace", gotQuery.Get("aggregate"))
	}
	if gotQuery.Get("includeIdle") != "true" || gotQuery.Get("shareIdle") != "true" {
		t.Errorf("idle flags wrong: %v", gotQuery)
	}
}
