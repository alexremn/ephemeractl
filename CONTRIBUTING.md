# Contributing to ephemeractl

Thanks for your interest in ephemeractl — *actual running cost of every PR's preview environment, posted on the PR.*

This guide covers local development, testing, and the contribution process.

## Scope

**v1 is the GitHub Action only**: read PR context → query OpenCost → render markdown → upsert one sticky comment. That is the whole surface area we accept changes against right now.

Out of scope for v1 (CRD/controller, TTL/orphan cleanup, scale-to-zero, dashboards, budgets/chargeback) is tracked in [docs/ROADMAP.md](docs/ROADMAP.md) and gated on adoption. Please do not open PRs implementing roadmap items before the gate they depend on has been met — file an issue to discuss first.

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

You need:

- **Go 1.22+** (`go version` to check)
- **git**
- Optional, for the "run locally" section: `kubectl` with access to a cluster running OpenCost, and `golangci-lint` for the full lint gate.

Clone and build:

```bash
git clone https://github.com/your-org/ephemeractl.git
cd ephemeractl
go build ./...
go test ./...
```

> Replace `your-org` with your fork's owner if you cloned a fork.

## Running the Action locally

The Action reads its configuration from `INPUT_*` environment variables (GitHub maps each `action.yml` input `foo-bar` to `INPUT_FOO-BAR`) and the PR event payload from the file at `$GITHUB_EVENT_PATH`. You can exercise it without GitHub Actions by setting those yourself.

### 1. Port-forward OpenCost

The default `opencost-url` resolves an in-cluster Service that a GitHub runner cannot reach from your laptop. Forward the OpenCost API (**port 9003** — the allocation API, *not* 9090) to localhost:

```bash
kubectl -n opencost port-forward svc/opencost 9003:9003
```

Sanity-check it returns allocation data:

```bash
curl 'http://localhost:9003/allocation?window=today&accumulate=true'
```

### 2. Provide a PR event payload

Point `GITHUB_EVENT_PATH` at a minimal `pull_request` event JSON containing at least the PR number, repo, owner, and `pull_request.created_at` (used to resolve the `pr-open` window). A trimmed payload pulled from a real PR with `gh api` works well for local runs.

### 3. Set inputs and run

```bash
export GITHUB_EVENT_PATH=/path/to/event.json
export INPUT_OPENCOST-URL=http://localhost:9003
export INPUT_PR-LABEL-KEY=ephemeractl.dev/pr
export INPUT_WINDOW=pr-open
export INPUT_IDLE-MODE=used-only
export INPUT_OPENCOST-RESOLUTION=1m
export INPUT_CURRENCY=USD
export INPUT_GITHUB-TOKEN=<a token with pull-requests:write>

go run ./cmd/ephemeractl
```

Leave `INPUT_GITHUB-TOKEN` unset (or use a read-only token) if you only want to observe the rendered markdown and do not want to post a real comment — printing the render and skipping the upsert is the recommended way to iterate on output.

## Code style

- **Formatting:** `gofmt` — run `gofmt -l .` and ensure it reports nothing. CI rejects unformatted code.
- **Vetting:** `go vet ./...` must pass.
- **Linting:** `golangci-lint run` must pass. Install it from <https://golangci-lint.run> and run it before pushing; CI runs the same.

Keep changes small and idiomatic Go. Follow the package boundaries in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) (`internal/ghevent`, `internal/opencost`, `internal/render`, `internal/comment`, `cmd/ephemeractl`); don't reach across them or fold unrelated concerns into one package.

## Tests

We require:

- **Table-driven tests** for each package, with descriptive subtests.
- **`net/http/httptest` mocks** for the OpenCost and GitHub APIs — no live cluster or live GitHub calls in the test suite.
- A **golden-file render test** for the markdown output, covering the total, the per-component breakdown, the per-team table, the window + idle line, and the honesty note. Update goldens deliberately (e.g. behind a `-update` flag), and review golden diffs in your PR.
- **≥80% coverage on `internal/*`.**

Run the suite the way CI does:

```bash
go test -race -cover ./...
```

When summing OpenCost allocation cost, note the correctness rule the tests pin down: there is **no `totalCost` and no flat `pvCost` field** in the allocation JSON. Cost is summed from components — `cpuCost + ramCost + gpuCost + networkCost + loadBalancerCost + sharedCost` plus the sum of `pvs[*].cost`. Don't "simplify" that away.

## Commits: Conventional Commits + DCO sign-off

Use [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `chore:`, `ci:`, `test:`, `refactor:`.

All commits must carry a Developer Certificate of Origin sign-off. Add it with `-s`:

```bash
git commit -s -m "feat: add per-team cost breakdown"
```

This appends a `Signed-off-by:` trailer asserting you have the right to submit the change under the project's Apache-2.0 license. CI checks for it.

## Pull request process

1. **Small, focused PRs.** One concern per PR; it gets reviewed and merged faster.
2. **CI must be green** — `go build`, `go vet`, `golangci-lint`, and `go test -race -cover` all pass, coverage holds at ≥80% on `internal/*`, and the DCO check passes.
3. Reference any related issue, and describe what you changed and how you verified it.
4. For anything touching the cost mechanism, honesty note, or `action.yml` inputs/outputs, keep it consistent with [docs/SPEC-cost-attribution.md](docs/SPEC-cost-attribution.md) and [docs/USAGE.md](docs/USAGE.md) — these are the contract.

> **Note:** `docs/superpowers/` is internal and gitignored. Do not add files there or reference it from public docs.

Questions or design discussion? Open an issue. Thanks for contributing.
