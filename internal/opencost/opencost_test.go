package opencost

import (
	"math"
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
