# Architecture

ephemeractl is a single small Go binary, shipped as a Docker container action. It
reads PR context from the GitHub event, queries OpenCost for the live cost of that
PR's preview environment, renders a markdown report, and upserts one sticky comment
on the PR.

> Tagline: Actual running cost of every PR's preview environment, posted on the PR.

The cost-attribution mechanism (selector, OpenCost query, component summing, the
honesty model, and the known sharp edges) is specified separately in
[SPEC-cost-attribution.md](./SPEC-cost-attribution.md). This document covers the
binary's structure, package responsibilities, dependencies, and runtime behavior.

## Data flow

```
GitHub event ($GITHUB_EVENT_PATH) ─► PR number, repo, owner, pull_request.created_at
INPUT_* env vars ─────────────────► selector, window, opencost-url, team-label,
                                     idle-mode, github-token, currency
        │
        ▼
  internal/ghevent   parse event payload + inputs → typed Config
        │
        ▼
  internal/opencost  build query, GET /allocation, sum components, group by team
        │
        ▼
  internal/render    markdown: total, breakdown table, per-team table,
                     window+idle line, honesty note
        │
        ▼
  internal/comment   find <!-- ephemeractl:cost-report --> marker
                     → Issues.EditComment else CreateComment
        │
        ▼
  cmd/ephemeractl    wiring + error handling + exit codes
```

The flow is strictly linear: each stage consumes the previous stage's typed output.
There is no shared mutable state and no controller loop — the binary runs once per
Action invocation and exits.

## Package responsibilities

| Package | Responsibility |
|---------|---------------|
| `internal/ghevent` | Parse the GitHub event payload at `$GITHUB_EVENT_PATH` and the `INPUT_*` env vars into a typed `Config` (owner, repo, PR number, `pull_request.created_at`, selector, window, OpenCost URL, team label, idle mode, resolution, currency, token). Validates inputs and resolves the selector (label vs namespace) and the `pr-open` window into the concrete query parameters. |
| `internal/opencost` | Build the `/allocation` query from the resolved `Config`, perform the `GET` over `net/http`, decode the JSON with `encoding/json`, sum cost components per allocation (`cpuCost + ramCost + gpuCost + networkCost + loadBalancerCost + sharedCost + Σ pvs[*].cost`), and group the result by team when `team-label` is set. Returns a typed cost result (total plus optional per-team groups). |
| `internal/render` | Render the cost result to markdown: the total, a per-component breakdown table, a per-team table (when grouped), a line stating the window and idle policy, and the honesty note. The first line of the body is always the marker `<!-- ephemeractl:cost-report -->`. |
| `internal/comment` | Upsert the sticky comment. List the PR's issue comments, find the one whose body begins with `<!-- ephemeractl:cost-report -->`; if found, call `Issues.EditComment`, otherwise `Issues.CreateComment`. Returns the resulting comment URL. |
| `cmd/ephemeractl` | Entry point. Wires the packages together, owns top-level error handling, writes the Action outputs (`total-cost`, `currency`, `comment-url`), and sets the process exit code. |

## Dependencies

- `github.com/google/go-github` (BSD-3-Clause) — GitHub REST client used by
  `internal/comment` to list, create, and edit PR comments.
- Standard library `net/http` + `encoding/json` — used by `internal/opencost` to
  call the OpenCost `/allocation` endpoint and decode the response. OpenCost has no
  auth by default, so no client SDK is required.

There is **no controller-runtime** and no Kubernetes client in v1. The binary never
talks to the Kubernetes API; it only reads the GitHub event, calls OpenCost over
HTTP, and calls the GitHub API. This keeps the image small and the dependency
surface minimal, consistent with the v1 scope (the Action only — no CRD/controller).

## Exit-code behavior

`cmd/ephemeractl` distinguishes hard failures from a graceful empty result:

- **Non-zero exit (hard failure).** Unreachable or erroring OpenCost endpoint,
  malformed or missing GitHub event, invalid configuration (e.g. neither selector
  resolvable, an unparseable window), or a GitHub API error while upserting the
  comment. These fail the workflow step so the problem is visible.
- **Zero exit with a "no data" comment (graceful degradation).** When OpenCost
  responds successfully but returns no allocations for the PR's selector and window,
  the binary posts a sticky comment saying no cost data was found for the PR and
  exits `0`. A PR with no measurable preview spend must not fail the build — the
  signal degrades gracefully rather than blocking the developer.

In both the populated and "no data" cases the comment is upserted via the same
marker logic, so a PR never accumulates duplicate cost comments.

## Why a Docker action with a prebuilt image

The Action is published as `runs.using: docker` with
`image: docker://ghcr.io/your-org/ephemeractl:<tag>` — a prebuilt image rather than
a Dockerfile built at runtime.

- **Fast start.** A prebuilt, pinned image is pulled and run directly; the workflow
  step does not build the container on every invocation.
- **Self-contained runtime.** The Go binary plus its single dependency ship inside
  the image, so the Action has no dependency on the runner's installed toolchain.
  This matters because the target deployment is a **self-hosted runner in or near the
  cluster** (so the in-cluster OpenCost Service URL resolves) — the image is the unit
  of distribution there.
- **Reproducibility.** Pinning the image tag (and the `your-org/ephemeractl@v1`
  action ref) gives callers a stable, auditable artifact.

The image and the `your-org` owner placeholder are documented for replacement in the
repository README and USAGE docs.

## Fork-PR limitation (documented, not solved in v1)

On a `pull_request` event from a fork, `GITHUB_TOKEN` is read-only with no access to
secrets, so the Action cannot post a comment. This is a documented v1 limitation. The
secure upgrade path is the `workflow_run` two-workflow pattern (a privileged workflow
triggered by the completed PR workflow). v1 does **not** implement this in code; it is
called out so users understand the boundary.
