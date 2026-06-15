// Package ghevent gathers the Action's runtime configuration from the
// INPUT_* environment variables and the GitHub event payload file.
package ghevent

import (
	"encoding/json"
	"fmt"
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

// eventPayload is the subset of a pull_request event we read.
type eventPayload struct {
	PullRequest *struct {
		Number    int       `json:"number"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"pull_request"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

// Load returns the full Config: env inputs plus PR identity read from
// $GITHUB_EVENT_PATH. It errors if the event is missing or is not a PR event.
func Load() (Config, error) {
	c := loadInputs()

	path := os.Getenv("GITHUB_EVENT_PATH")
	if path == "" {
		return c, fmt.Errorf("GITHUB_EVENT_PATH is not set; the Action must run on a pull_request event")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("read event payload %q: %w", path, err)
	}
	var ev eventPayload
	if err := json.Unmarshal(raw, &ev); err != nil {
		return c, fmt.Errorf("parse event payload: %w", err)
	}
	if ev.PullRequest == nil {
		return c, fmt.Errorf("event payload has no pull_request; ephemeractl runs on pull_request events")
	}
	c.Owner = ev.Repository.Owner.Login
	c.Repo = ev.Repository.Name
	c.PRNumber = ev.PullRequest.Number
	c.PRCreatedAt = ev.PullRequest.CreatedAt
	return c, nil
}
