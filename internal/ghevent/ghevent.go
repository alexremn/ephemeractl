// Package ghevent gathers the Action's runtime configuration from the
// INPUT_* environment variables and the GitHub event payload file.
package ghevent

import (
	"os"
	"time"
)

// Config is the fully-resolved input to a single Action run.
type Config struct {
	OpenCostURL      string
	PRLabelKey       string
	NamespacePattern string
	Window           string
	TeamLabel        string
	IdleMode         string // "used-only" | "include-idle"
	Resolution       string
	Currency         string
	GitHubToken      string

	// Derived from the event payload:
	Owner       string
	Repo        string
	PRNumber    int
	PRCreatedAt time.Time
}

// input reads INPUT_<NAME> (GitHub's convention for action inputs), falling
// back to def when unset or empty.
func input(name, def string) string {
	if v := os.Getenv("INPUT_" + name); v != "" {
		return v
	}
	return def
}

// loadInputs reads only the env-derived fields (no event parsing).
func loadInputs() Config {
	return Config{
		OpenCostURL:      input("OPENCOST-URL", "http://opencost.opencost.svc.cluster.local:9003"),
		PRLabelKey:       input("PR-LABEL-KEY", "ephemeractl.dev/pr"),
		NamespacePattern: input("NAMESPACE-PATTERN", ""),
		Window:           input("WINDOW", "pr-open"),
		TeamLabel:        input("TEAM-LABEL", ""),
		IdleMode:         input("IDLE-MODE", "used-only"),
		Resolution:       input("OPENCOST-RESOLUTION", "1m"),
		Currency:         input("CURRENCY", "USD"),
		GitHubToken:      input("GITHUB-TOKEN", ""),
	}
}
