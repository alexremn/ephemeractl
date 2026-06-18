// Package opencost queries the OpenCost Allocation API and reduces it to a
// single PR's cost. OpenCost returns no totalCost/pvCost field, so cost is
// summed from components here. See docs/SPEC-cost-attribution.md.
package opencost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// pvCost is one entry of an allocation's "pvs" map.
type pvCost struct {
	Cost float64 `json:"cost"`
}

// Allocation mirrors the cost-bearing fields of an OpenCost allocation.
type Allocation struct {
	Name             string            `json:"name"`
	CPUCost          float64           `json:"cpuCost"`
	RAMCost          float64           `json:"ramCost"`
	GPUCost          float64           `json:"gpuCost"`
	NetworkCost      float64           `json:"networkCost"`
	LoadBalancerCost float64           `json:"loadBalancerCost"`
	SharedCost       float64           `json:"sharedCost"`
	PVs              map[string]pvCost `json:"pvs"`
}

// Total sums every cost component, including persistent-volume costs.
func (a Allocation) Total() float64 {
	t := a.CPUCost + a.RAMCost + a.GPUCost + a.NetworkCost + a.LoadBalancerCost + a.SharedCost
	for _, pv := range a.PVs {
		t += pv.Cost
	}
	return t
}

// Components is the per-resource cost breakdown for a whole query result.
type Components struct {
	CPU          float64
	RAM          float64
	GPU          float64
	Network      float64
	LoadBalancer float64
	Shared       float64
	PV           float64
}

// Group is one aggregation bucket (a team-label value or a namespace).
type Group struct {
	Name string
	Cost float64
}

// Result is the reduced cost for a single PR.
type Result struct {
	Total      float64
	Components Components
	Groups     []Group
}

// Query describes one allocation request.
type Query struct {
	Window      string
	Resolution  string
	LabelKey    string // selector: PR label key (label mode)
	LabelValue  string // selector: PR number as string (label mode)
	Namespace   string // selector: resolved namespace (namespace mode)
	TeamLabel   string // aggregate by this label; empty → aggregate=namespace
	IncludeIdle bool
}

// allocationPath is the OpenCost Allocation API endpoint path.
const allocationPath = "/allocation"

// idleKey is the sentinel name OpenCost uses for the idle allocation bucket.
const idleKey = "__idle__"

// Client talks to one OpenCost Allocation API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New returns a Client for the given OpenCost base URL (e.g.
// http://opencost.opencost.svc.cluster.local:9003).
func New(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// maxResponseBytes caps how much of the OpenCost response body is read into
// memory, guarding against an unexpectedly large payload.
const maxResponseBytes = 16 << 20 // 16 MiB

type apiResponse struct {
	Code int                     `json:"code"`
	Data []map[string]Allocation `json:"data"`
}

// Fetch queries /allocation, sums components, and returns the reduced Result.
func (c *Client) Fetch(ctx context.Context, q Query) (Result, error) {
	qs := url.Values{}
	qs.Set("window", q.Window)
	qs.Set("accumulate", "true")
	if q.Resolution != "" {
		qs.Set("resolution", q.Resolution)
	}
	if q.Namespace != "" {
		qs.Set("filterNamespaces", q.Namespace)
	} else {
		qs.Set("filterLabels", q.LabelKey+":"+q.LabelValue)
	}
	if q.TeamLabel != "" {
		qs.Set("aggregate", "label:"+q.TeamLabel)
	} else {
		qs.Set("aggregate", "namespace")
	}
	if q.IncludeIdle {
		qs.Set("includeIdle", "true")
		qs.Set("shareIdle", "true")
	}

	endpoint := c.baseURL + allocationPath + "?" + qs.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("reach OpenCost at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("OpenCost returned HTTP %d", resp.StatusCode)
	}

	var ar apiResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&ar); err != nil {
		return Result{}, fmt.Errorf("decode OpenCost response: %w", err)
	}
	return reduce(ar), nil
}

// reduce collapses the (accumulated) allocation set into a Result.
func reduce(ar apiResponse) Result {
	var res Result
	if len(ar.Data) == 0 {
		return res
	}
	for name, a := range ar.Data[0] {
		if name == idleKey {
			continue // idle is shared into groups when requested; never its own group
		}
		cost := a.Total()
		res.Total += cost
		res.Components.CPU += a.CPUCost
		res.Components.RAM += a.RAMCost
		res.Components.GPU += a.GPUCost
		res.Components.Network += a.NetworkCost
		res.Components.LoadBalancer += a.LoadBalancerCost
		res.Components.Shared += a.SharedCost
		for _, pv := range a.PVs {
			res.Components.PV += pv.Cost
		}
		res.Groups = append(res.Groups, Group{Name: name, Cost: cost})
	}
	sort.Slice(res.Groups, func(i, j int) bool {
		if res.Groups[i].Cost == res.Groups[j].Cost {
			return res.Groups[i].Name < res.Groups[j].Name
		}
		return res.Groups[i].Cost > res.Groups[j].Cost
	})
	return res
}
