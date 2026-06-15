// Package opencost queries the OpenCost Allocation API and reduces it to a
// single PR's cost. OpenCost returns no totalCost/pvCost field, so cost is
// summed from components here. See docs/SPEC-cost-attribution.md.
package opencost

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
