// Package render turns a reduced cost Result into the sticky PR comment.
package render

import (
	"fmt"
	"strings"

	"github.com/alexremn/ephemeractl/internal/opencost"
)

// Marker is the hidden HTML comment that identifies ephemeractl's comment so
// it can be updated in place. It MUST be the first line of every comment.
const Marker = "<!-- ephemeractl:cost-report -->"

// Report is everything render needs to produce the comment.
type Report struct {
	PRNumber   int
	Total      float64
	Currency   string
	Window     string
	IdleMode   string
	Components opencost.Components
	Groups     []opencost.Group
}

const honesty = "> 💸 Approximate **lower bound** from OpenCost on-demand list rates — excludes " +
	"spot/RI/committed-use discounts; network egress is 0 unless the egress DaemonSet is enabled; " +
	"leaked/unmounted PV and some load-balancer cost may be undercounted. Use for relative signal " +
	"and trend, not invoice reconciliation."

// Markdown renders the comment body. It always begins with Marker.
func Markdown(r Report) string {
	var b strings.Builder
	b.WriteString(Marker + "\n")
	fmt.Fprintf(&b, "### Preview environment cost — PR #%d\n\n", r.PRNumber)

	hasData := r.Total > 0 || len(r.Groups) > 0
	if !hasData {
		fmt.Fprintf(&b, "No cost data found for this PR yet (window: %s).\n", r.Window)
		b.WriteString("If the environment is running, check that its pods carry the PR label on the ")
		b.WriteString("pod template — see the troubleshooting section of `docs/USAGE.md`.\n\n")
		b.WriteString(honesty + "\n")
		return b.String()
	}

	fmt.Fprintf(&b, "**Total: %s %.2f** (window: %s · idle-mode: %s)\n\n",
		r.Currency, r.Total, r.Window, r.IdleMode)

	b.WriteString("| Resource | Cost |\n|---|--:|\n")
	rows := []struct {
		label string
		val   float64
	}{
		{"CPU", r.Components.CPU},
		{"Memory", r.Components.RAM},
		{"GPU", r.Components.GPU},
		{"Network", r.Components.Network},
		{"Load balancer", r.Components.LoadBalancer},
		{"Storage (PV)", r.Components.PV},
		{"Shared", r.Components.Shared},
	}
	for _, row := range rows {
		if row.val > 0 {
			fmt.Fprintf(&b, "| %s | %s %.2f |\n", row.label, r.Currency, row.val)
		}
	}
	fmt.Fprintf(&b, "| **Total** | **%s %.2f** |\n\n", r.Currency, r.Total)

	if len(r.Groups) > 0 {
		b.WriteString("**By team**\n\n| Team | Cost |\n|---|--:|\n")
		for _, g := range r.Groups {
			name := g.Name
			if name == "__unallocated__" {
				name = "untagged"
			}
			fmt.Fprintf(&b, "| %s | %s %.2f |\n", cell(name), r.Currency, g.Cost)
		}
		b.WriteString("\n")
	}

	b.WriteString(honesty + "\n")
	return b.String()
}

// cell escapes a value so OpenCost-supplied text cannot break out of a markdown
// table cell.
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
