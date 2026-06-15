package ghevent

import "testing"

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
