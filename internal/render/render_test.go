package render

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/ephemeractl/internal/opencost"
)

var update = flag.Bool("update", false, "update golden files")

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v (run with -update to create)", path, err)
	}
	if got != string(want) {
		t.Errorf("rendered output != %s\n--- got ---\n%s", name, got)
	}
}

func TestMarkdownReport(t *testing.T) {
	out := Markdown(Report{
		PRNumber: 482,
		Total:    4.17,
		Currency: "USD",
		Window:   "pr-open",
		IdleMode: "used-only",
		Components: opencost.Components{
			CPU: 2.10, RAM: 1.20, Network: 0.30, LoadBalancer: 0.25, PV: 0.32,
		},
		Groups: []opencost.Group{
			{Name: "checkout", Cost: 2.50},
			{Name: "payments", Cost: 1.67},
		},
	})
	if out[:len(Marker)] != Marker {
		t.Errorf("output must start with the sticky marker")
	}
	checkGolden(t, "report.golden", out)
}

func TestMarkdownNoData(t *testing.T) {
	out := Markdown(Report{PRNumber: 9, Currency: "USD", Window: "pr-open", IdleMode: "used-only"})
	checkGolden(t, "no_data.golden", out)
}
