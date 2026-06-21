// Command ephemeractl posts the actual running cost of a PR's preview
// environment as a sticky comment on the PR. See docs/ for the full design.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alexremn/ephemeractl/internal/comment"
	"github.com/alexremn/ephemeractl/internal/ghevent"
	"github.com/alexremn/ephemeractl/internal/opencost"
	"github.com/alexremn/ephemeractl/internal/render"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := ghevent.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ephemeractl: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	window := resolveWindow(cfg, time.Now().UTC())
	query := resolveQuery(cfg, window)

	res, err := opencost.New(cfg.OpenCostURL).Fetch(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ephemeractl: %v\n", err)
		return 1
	}

	// Only break cost down by team when a team-label was configured; otherwise
	// OpenCost aggregates by namespace and a "By team" table would be misleading.
	var groups []opencost.Group
	if cfg.TeamLabel != "" {
		groups = res.Groups
	}

	body := render.Markdown(render.Report{
		PRNumber:   cfg.PRNumber,
		Total:      res.Total,
		Currency:   cfg.Currency,
		Window:     cfg.Window,
		IdleMode:   cfg.IdleMode,
		Components: res.Components,
		Groups:     groups,
	})

	url, err := comment.NewPoster(cfg.GitHubToken, cfg.Owner, cfg.Repo, cfg.PRNumber).
		Upsert(ctx, render.Marker, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ephemeractl: %v\n", err)
		return 1
	}

	if err := writeOutputs(res.Total, cfg.Currency, url); err != nil {
		fmt.Fprintf(os.Stderr, "ephemeractl: %v\n", err)
		return 1
	}
	fmt.Printf("ephemeractl: posted cost %.2f %s for PR #%d → %s\n",
		res.Total, cfg.Currency, cfg.PRNumber, url)
	return 0
}

// resolveWindow turns the "pr-open" token into a concrete created_at,now
// RFC3339 range; any other value is passed through to OpenCost unchanged.
func resolveWindow(cfg ghevent.Config, now time.Time) string {
	if cfg.Window != "pr-open" {
		return cfg.Window
	}
	return cfg.PRCreatedAt.UTC().Format(time.RFC3339) + "," + now.UTC().Format(time.RFC3339)
}

// resolveQuery maps the resolved config to an opencost.Query, choosing the
// namespace selector when a pattern is set, else the PR-label selector.
func resolveQuery(cfg ghevent.Config, window string) opencost.Query {
	q := opencost.Query{
		Window:      window,
		Resolution:  cfg.Resolution,
		TeamLabel:   cfg.TeamLabel,
		IncludeIdle: cfg.IdleMode == "include-idle",
	}
	if cfg.NamespacePattern != "" {
		q.Namespace = strings.ReplaceAll(cfg.NamespacePattern, "{pr}", strconv.Itoa(cfg.PRNumber))
	} else {
		q.LabelKey = cfg.PRLabelKey
		q.LabelValue = strconv.Itoa(cfg.PRNumber)
	}
	return q
}

// writeOutputs appends the Action outputs to the $GITHUB_OUTPUT file.
func writeOutputs(total float64, currency, commentURL string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil // not running under Actions; nothing to write
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600) // #nosec G304 G703 -- path is $GITHUB_OUTPUT set by the Actions runner, not user input
	if err != nil {
		return fmt.Errorf("open GITHUB_OUTPUT: %w", err)
	}
	if _, err := fmt.Fprintf(f, "total-cost=%.2f\ncurrency=%s\ncomment-url=%s\n", total, currency, commentURL); err != nil {
		_ = f.Close()
		return fmt.Errorf("write GITHUB_OUTPUT: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close GITHUB_OUTPUT: %w", err)
	}
	return nil
}
