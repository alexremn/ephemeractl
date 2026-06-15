package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/your-org/ephemeractl/internal/ghevent"
	"github.com/your-org/ephemeractl/internal/opencost"
)

func TestResolveWindowPROpen(t *testing.T) {
	cfg := ghevent.Config{
		Window:      "pr-open",
		PRCreatedAt: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	got := resolveWindow(cfg, now)
	want := "2026-06-01T10:00:00Z,2026-06-02T10:00:00Z"
	if got != want {
		t.Errorf("resolveWindow = %q, want %q", got, want)
	}
}

func TestResolveWindowPassthrough(t *testing.T) {
	cfg := ghevent.Config{Window: "7d"}
	if got := resolveWindow(cfg, time.Now()); got != "7d" {
		t.Errorf("resolveWindow = %q, want 7d", got)
	}
}

func TestResolveQueryLabelMode(t *testing.T) {
	cfg := ghevent.Config{
		PRLabelKey: "ephemeractl.dev/pr", PRNumber: 482,
		TeamLabel: "team", IdleMode: "used-only", Resolution: "1m", Window: "7d",
	}
	q := resolveQuery(cfg, "7d")
	want := opencost.Query{
		Window: "7d", Resolution: "1m",
		LabelKey: "ephemeractl.dev/pr", LabelValue: "482",
		TeamLabel: "team", IncludeIdle: false,
	}
	if q != want {
		t.Errorf("resolveQuery = %+v, want %+v", q, want)
	}
}

func TestResolveQueryNamespaceModeAndIdle(t *testing.T) {
	cfg := ghevent.Config{
		NamespacePattern: "preview-pr-{pr}", PRNumber: 7,
		IdleMode: "include-idle", Resolution: "1m", Window: "7d",
	}
	q := resolveQuery(cfg, "7d")
	if q.Namespace != "preview-pr-7" || q.LabelKey != "" || !q.IncludeIdle {
		t.Errorf("namespace mode wrong: %+v", q)
	}
}

func TestWriteOutputs(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", f.Name())
	if err := writeOutputs(4.17, "USD", "https://example/c"); err != nil {
		t.Fatalf("writeOutputs: %v", err)
	}
	data, _ := os.ReadFile(f.Name())
	s := string(data)
	if !strings.Contains(s, "total-cost=4.17") || !strings.Contains(s, "currency=USD") ||
		!strings.Contains(s, "comment-url=https://example/c") {
		t.Errorf("outputs missing: %q", s)
	}
}
