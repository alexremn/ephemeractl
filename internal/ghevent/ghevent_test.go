package ghevent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadInputsAppliesDefaults(t *testing.T) {
	t.Setenv("INPUT_OPENCOST-URL", "http://opencost.opencost.svc.cluster.local:9003")
	t.Setenv("INPUT_GITHUB-TOKEN", "tok")
	// leave the rest unset → defaults

	c := loadInputs()

	if c.PRLabelKey != "ephemeractl.dev/pr" {
		t.Errorf("PRLabelKey = %q, want default", c.PRLabelKey)
	}
	if c.Window != "pr-open" {
		t.Errorf("Window = %q, want pr-open", c.Window)
	}
	if c.IdleMode != "used-only" {
		t.Errorf("IdleMode = %q, want used-only", c.IdleMode)
	}
	if c.Resolution != "1m" {
		t.Errorf("Resolution = %q, want 1m", c.Resolution)
	}
	if c.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", c.Currency)
	}
	if c.OpenCostURL == "" || c.GitHubToken != "tok" {
		t.Errorf("explicit inputs not read: url=%q token=%q", c.OpenCostURL, c.GitHubToken)
	}
}

func TestLoadInputsOverrides(t *testing.T) {
	t.Setenv("INPUT_OPENCOST-URL", "http://oc:9003")
	t.Setenv("INPUT_PR-LABEL-KEY", "custom/pr")
	t.Setenv("INPUT_NAMESPACE-PATTERN", "preview-pr-{pr}")
	t.Setenv("INPUT_WINDOW", "7d")
	t.Setenv("INPUT_TEAM-LABEL", "team")
	t.Setenv("INPUT_IDLE-MODE", "include-idle")
	t.Setenv("INPUT_CURRENCY", "EUR")

	c := loadInputs()

	if c.PRLabelKey != "custom/pr" || c.NamespacePattern != "preview-pr-{pr}" ||
		c.Window != "7d" || c.TeamLabel != "team" || c.IdleMode != "include-idle" ||
		c.Currency != "EUR" {
		t.Errorf("overrides not applied: %+v", c)
	}
}

func TestLoadParsesEvent(t *testing.T) {
	dir := t.TempDir()
	eventPath := filepath.Join(dir, "event.json")
	payload := `{
	  "pull_request": { "number": 482, "created_at": "2026-06-01T10:00:00Z" },
	  "repository": { "name": "checkout", "owner": { "login": "acme" } }
	}`
	if err := os.WriteFile(eventPath, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	t.Setenv("INPUT_OPENCOST-URL", "http://oc:9003")
	t.Setenv("INPUT_GITHUB-TOKEN", "tok")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if c.Owner != "acme" || c.Repo != "checkout" || c.PRNumber != 482 {
		t.Errorf("event fields wrong: owner=%q repo=%q pr=%d", c.Owner, c.Repo, c.PRNumber)
	}
	want := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	if !c.PRCreatedAt.Equal(want) {
		t.Errorf("PRCreatedAt = %v, want %v", c.PRCreatedAt, want)
	}
}

func TestLoadErrorsWithoutPR(t *testing.T) {
	dir := t.TempDir()
	eventPath := filepath.Join(dir, "event.json")
	if err := os.WriteFile(eventPath, []byte(`{"repository":{"name":"x","owner":{"login":"y"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	t.Setenv("INPUT_OPENCOST-URL", "http://oc:9003")

	if _, err := Load(); err == nil {
		t.Fatal("expected error when event has no pull_request, got nil")
	}
}
