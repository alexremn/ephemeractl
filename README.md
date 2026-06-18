# ephemeractl

> Actual running cost of every PR's preview environment, posted on the PR.

[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](https://go.dev)
[![status: v1](https://img.shields.io/badge/status-v1%20%C2%B7%20validate--first-brightgreen.svg)](docs/ROADMAP.md)
[![CI](https://github.com/alexremn/ephemeractl/actions/workflows/ci.yml/badge.svg)](https://github.com/alexremn/ephemeractl/actions/workflows/ci.yml)

## What it does

ephemeractl is a self-hostable **GitHub Action**. On a pull request it queries
[OpenCost](https://www.opencost.io/) for the **actual running cost so far** of that PR's
Kubernetes preview environment, then posts and keeps updating a single sticky comment:

```markdown
<!-- ephemeractl:cost-report -->
### Preview environment cost — PR #482

**Total: USD 4.17** (window: pr-open · idle-mode: used-only)

| Resource | Cost |
|---|--:|
| CPU | USD 2.10 |
| Memory | USD 1.20 |
| Network | USD 0.30 |
| Load balancer | USD 0.25 |
| Storage (PV) | USD 0.32 |
| **Total** | **USD 4.17** |

**By team**

| Team | Cost |
|---|--:|
| checkout | USD 2.50 |
| payments | USD 1.67 |

> 💸 Approximate **lower bound** from OpenCost on-demand list rates — excludes spot/RI/committed-use discounts; network egress is 0 unless the egress DaemonSet is enabled; leaked/unmounted PV and some load-balancer cost may be undercounted. Use for relative signal and trend, not invoice reconciliation.
```

The first line is the HTML marker `<!-- ephemeractl:cost-report -->`. ephemeractl finds it to
update the existing comment in place instead of posting a new one on every push. The **By team**
table appears only when you set `team-label` (see [Configuration](#configuration)); the quickstart
below omits it and shows a single total.

## Why

Per-PR preview environments are already a solved commodity: ArgoCD's ApplicationSet PR generator
plus kube-janitor handle create, teardown, and TTL. The unmet slice is **cost** — teams get one
opaque monthly bill and no per-PR signal. Existing "cost on PR" tools (Kubecost's
`cost-prediction-action`, Infracost) price the **declared manifest before apply** — a *prediction*.
ephemeractl reports the **actual running spend** of the live environment pulled from OpenCost. That
"actual vs predicted" gap is the entire point.

## How it works

ephemeractl reads the PR context from the GitHub event, resolves a selector (a per-PR pod-template
label by default, or a namespace pattern), queries the OpenCost `/allocation` API for the window,
sums the cost components itself (OpenCost has no flat total), renders the markdown above, and
upserts the sticky comment. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the components and
data flow, and [docs/SPEC-cost-attribution.md](docs/SPEC-cost-attribution.md) for the selector,
query, cost-summing, and honesty model.

## Quickstart (60s)

**Prerequisites:**

- **OpenCost** installed in-cluster (Apache-2.0, CNCF). Its API listens on
  `…:9003/allocation` with no auth by default.
- **The OpenCost API is reachable from the runner.** The Action runs on a GitHub runner, so that
  runner must reach OpenCost. The normal setup is a **self-hosted runner in or near the cluster**
  (the default `opencost-url` resolves the in-cluster Service). Hosted runners need an
  ingress/tunnel URL. This is a hard prerequisite.
- **Preview-env workloads labelled per PR.** Your preview environments must carry an immutable label
  whose value is the PR number on the **pod template** (`spec.template.metadata.labels`), default
  key `ephemeractl.dev/pr`. See the [ArgoCD ApplicationSet example](examples/argocd-applicationset.yaml).

Minimal workflow:

```yaml
name: preview-cost
on:
  pull_request:

permissions:
  pull-requests: write

jobs:
  cost:
    runs-on: [self-hosted]   # must reach the OpenCost API
    steps:
      - uses: alexremn/ephemeractl@v1
        with:
          # opencost-url defaults to the in-cluster Service; override for ingress/tunnel.
          pr-label-key: ephemeractl.dev/pr
          window: pr-open
```

The default `github.token` is sufficient; `permissions: pull-requests: write` is required so the
Action can create and update the sticky comment.

Full setup, the ArgoCD label how-to, and the verify step are in
[docs/USAGE.md](docs/USAGE.md); runnable files are in [examples/](examples/).

## Configuration

| Input | Default | Purpose |
|-------|---------|---------|
| `opencost-url` | `http://opencost.opencost.svc.cluster.local:9003` | OpenCost API base; override for ingress/tunnel |
| `pr-label-key` | `ephemeractl.dev/pr` | Pod-template label carrying the PR number (label-selector mode) |
| `namespace-pattern` | _(empty)_ | Alternative selector, e.g. `preview-pr-{pr}`; overrides label mode when set |
| `window` | `pr-open` | `pr-open` (created_at → now) or any OpenCost window |
| `team-label` | _(empty)_ | Label to break cost down by team; empty → single total |
| `idle-mode` | `used-only` | `used-only` or `include-idle` |
| `opencost-resolution` | `1m` | OpenCost query resolution |
| `currency` | `USD` | Display symbol/code only (OpenCost returns plain numbers) |
| `github-token` | `${{ github.token }}` | Token for the sticky comment |

**Outputs:** `total-cost` (number), `currency`, `comment-url`.

## Accuracy & honesty

The reported figure is an **approximate lower bound**, by design:

- OpenCost on-demand **list rates only** — no spot, reserved-instance, or committed-use
  reconciliation.
- **Network egress counts as 0** unless OpenCost's egress DaemonSet is enabled.
- **Leaked/unmounted PVs and some load-balancer cost** may be undercounted (they live in OpenCost
  *Assets*, not Allocation).
- Idle/shared cost is a **policy choice** exposed as `idle-mode` (default `used-only`), because bare
  OpenCost has no `shareNamespaces`/`shareCost` knobs.

Treat the number as a trustworthy **relative signal and trend**, not invoice reconciliation. This
honesty is what earns adoption — the details are in
[docs/SPEC-cost-attribution.md](docs/SPEC-cost-attribution.md).

**Fork PRs:** on `pull_request` from a fork, `GITHUB_TOKEN` is read-only with no secrets, so the
Action cannot comment. This is a documented limitation in v1; the secure upgrade path is the
`workflow_run` two-workflow pattern (see [docs/USAGE.md](docs/USAGE.md)). It is **not** solved in v1
code.

## Roadmap

v1 is the GitHub Action only; everything heavier is gated on it proving adoption. See
[docs/ROADMAP.md](docs/ROADMAP.md).

---

[Contributing](CONTRIBUTING.md) · [Security](SECURITY.md) · [Code of Conduct](CODE_OF_CONDUCT.md) · [License (Apache-2.0)](LICENSE)
