# Usage

Operator guide for **ephemeractl** — *Actual running cost of every PR's preview environment, posted on the PR.*

ephemeractl is a GitHub Action. It reads the PR context, queries OpenCost for the **actual running cost so far** of that PR's Kubernetes preview environment, and upserts a single sticky comment on the PR. This guide covers the prerequisites, the four setup steps, the full configuration reference, and troubleshooting.

> The cost figure is an **approximate lower bound**, not an invoice. See [Honesty model](#honesty-model) before you act on the numbers.

---

## Prerequisites

Three things must be true before the Action can report anything.

### 1. OpenCost is installed in-cluster

ephemeractl reads cost data from [OpenCost](https://www.opencost.io/) (Apache-2.0, CNCF). Install it following the [OpenCost installation guide](https://www.opencost.io/docs/installation/install).

- The OpenCost **API** serves allocation data at `…:9003/allocation`. This is the port the Action talks to. Port `9003` is the API; port `9090` is the OpenCost UI — **do not** point ephemeractl at `9090` (see [Troubleshooting](#troubleshooting)).
- By default OpenCost runs with **no auth**. The default `opencost-url` resolves the in-cluster Service: `http://opencost.opencost.svc.cluster.local:9003`.

### 2. The OpenCost API is reachable from where the workflow runs

The Action runs on a GitHub runner, and that runner must be able to reach the OpenCost API. This is a hard prerequisite.

- **Normal setup: a self-hosted runner in or near the cluster.** With a runner on the cluster network, the default in-cluster Service URL resolves and no extra wiring is needed.
- **Hosted runners** cannot reach an in-cluster Service. You must expose OpenCost through an ingress or a tunnel and set `opencost-url` to that reachable address. If you do, secure it — OpenCost ships with no auth.

### 3. Preview-env workloads are labeled per PR

ephemeractl finds "this PR's" workloads with a selector. The default and most robust selector is a label whose value is the PR number. There is no universal convention, so you label your workloads and ephemeractl is configurable around them. See [Step 1](#step-1-label-preview-workloads).

---

## Setup

### Step 1: Label preview workloads

The Action's default selector is the **label** `ephemeractl.dev/pr`, whose value is the PR number (configurable via `pr-label-key`).

**The label MUST sit on the pod template** — `spec.template.metadata.labels` — not only on the Deployment's top-level `metadata.labels`. OpenCost attributes cost by the labels on the running pods; a label that exists only on the Deployment object is invisible to it, and the cost lands in `__unallocated__`.

```yaml
# Deployment (excerpt) — the PR label must reach the pod template:
spec:
  template:
    metadata:
      labels:
        ephemeractl.dev/pr: "1234"   # value = PR number, on spec.template.metadata.labels
```

For a complete, working PR-generator setup that templates `ephemeractl.dev/pr` onto the pod spec for every preview environment, see [`examples/argocd-applicationset.yaml`](../examples/argocd-applicationset.yaml).

> **Alternative selector — namespace.** If you map each PR to its own namespace (e.g. `preview-pr-{pr}`), set `namespace-pattern` instead of labeling pods. It overrides label mode when set, but it breaks on multi-namespace environments and namespace reuse, so the label selector is the default. See the [configuration reference](#configuration-reference).

### Step 2: Verify the label is visible to OpenCost

Before you trust any number, confirm OpenCost actually sees your PR label. Prometheus and kube-state-metrics (KSM) sanitize label keys (`ephemeractl.dev/pr` becomes `label_ephemeractl_dev_pr` internally), and KSM may not export your label at all unless it is on its allowlist.

Run a sample query against the OpenCost API — aggregate the last hour by your PR label and check that your PR number shows up as its own group rather than collapsing into `__unallocated__`:

```bash
curl -s "http://opencost.opencost.svc.cluster.local:9003/allocation?window=1h&aggregate=label:ephemeractl.dev/pr&accumulate=true"
```

In the response, look for a key matching your PR number (e.g. `"1234"`). If you only see `"__unallocated__"`, OpenCost does not yet see the label:

- Confirm the label is on `spec.template.metadata.labels` (Step 1), not just the Deployment.
- Confirm KSM is exporting it — you may need `--metric-labels-allowlist` to include your label key. See [Troubleshooting](#troubleshooting).

Do not proceed until your PR appears as its own group.

### Step 3: Add the workflow

Add the consuming workflow to your repository. A ready-to-copy file is provided at [`examples/workflow-cost.yml`](../examples/workflow-cost.yml).

The minimum the caller must provide:

```yaml
permissions:
  pull-requests: write   # required — the Action upserts the sticky comment

jobs:
  cost:
    runs-on: [self-hosted]   # a runner that can reach the OpenCost API (see Prerequisites)
    steps:
      - uses: your-org/ephemeractl@v1
        with:
          opencost-url: http://opencost.opencost.svc.cluster.local:9003
          pr-label-key: ephemeractl.dev/pr
          # window, team-label, idle-mode, etc. — see configuration reference
```

`permissions: pull-requests: write` is required: without it the Action cannot create or edit the comment. The runner must satisfy the [network reachability prerequisite](#2-the-opencost-api-is-reachable-from-where-the-workflow-runs).

### Step 4: Understand the fork-PR limitation

ephemeractl **cannot comment on pull requests opened from a fork.**

On a `pull_request` event triggered by a fork, GitHub gives the workflow a **read-only `GITHUB_TOKEN` with no access to secrets**. The Action has no permission to write the comment, so it cannot post. This is a GitHub security boundary, not a bug in ephemeractl, and v1 does not work around it in code.

**Secure upgrade path — the `workflow_run` two-workflow pattern.** Split the work into two workflows:

1. A workflow on `pull_request` runs the cost computation (or just records the PR context) and uploads its result as an artifact. It needs no write permission and no secrets.
2. A second workflow on `workflow_run` (triggered by the first completing) runs in the **base repository's** trusted context, where it has a writable token, and posts the comment.

This is the standard, secure pattern for acting on fork PRs. It is documented here as the recommended path; it is **not** implemented for you in v1.

---

## Configuration reference

All inputs map to the Action's `with:` block. Defaults match the canonical action interface.

| Input | Default | Purpose |
|-------|---------|---------|
| `opencost-url` | `http://opencost.opencost.svc.cluster.local:9003` | OpenCost API base URL. Override for ingress/tunnel when not in-cluster. Port `9003` is the API. |
| `pr-label-key` | `ephemeractl.dev/pr` | Pod-template label key whose value is the PR number (label-selector mode). |
| `namespace-pattern` | _(empty)_ | Alternative selector, e.g. `preview-pr-{pr}`. Overrides label mode when set. |
| `window` | `pr-open` | `pr-open` (PR `created_at` → now, the env's lifetime spend) or any native OpenCost window (`7d`, `today`, `lastweek`, an RFC3339 pair, a unix pair). |
| `team-label` | _(empty)_ | Label to break cost down by team (e.g. `team`). Empty → single rolled-up total. |
| `idle-mode` | `used-only` | `used-only` or `include-idle`. Whether idle/shared capacity is included. |
| `opencost-resolution` | `1m` | OpenCost query resolution. |
| `currency` | `USD` | Display symbol/code only. OpenCost returns plain numbers. |
| `github-token` | `${{ github.token }}` | Token used to upsert the sticky comment. |

**Outputs:**

| Output | Description |
|--------|-------------|
| `total-cost` | Total cost for the PR over the resolved window (number). |
| `currency` | The currency code, echoing the `currency` input. |
| `comment-url` | URL of the sticky PR comment that was created or updated. |

The sticky comment is identified by the HTML marker `<!-- ephemeractl:cost-report -->` on its first line; the Action edits that comment in place rather than posting a new one each run.

### Window notes

- `pr-open` resolves to the RFC3339 pair `{pull_request.created_at},{now}` — the lifetime spend of the environment.
- Any window is bounded by your **Prometheus retention** (commonly ~15d unless extended). A `window` reaching past retention silently truncates.
- Prefer the immutable label selector and always time-bound the window: recycled PR numbers and reused namespaces will otherwise conflate history.

---

## Honesty model

Every comment states the figure is an **approximate lower bound**. Treat it as a trustworthy *relative* signal and trend, **not invoice reconciliation**:

- **On-demand list rates only.** OpenCost prices at on-demand list rates — no spot, reserved-instance, or committed-use discounts are reconciled, so real billed cost is typically lower.
- **Network egress is `0`** unless the OpenCost egress DaemonSet is enabled.
- **Leaked/unmounted PVs and some load-balancer cost** live in OpenCost *Assets*, not Allocation, and may be undercounted.
- **GPU** needs explicit pricing configuration to be counted.
- **Very short-lived pods** can be under-sampled at `1m` resolution.
- **Idle/shared inclusion is a policy choice** exposed via `idle-mode` (default `used-only`). Bare OpenCost has no `shareNamespaces`/`shareCost` knobs, so `used-only` is the honest default.

---

## Troubleshooting

### No data / cost shows as `__unallocated__`

The label is not reaching OpenCost.

- The PR label is on the Deployment only, not on `spec.template.metadata.labels`. Move it to the pod template (Step 1).
- KSM is not exporting your label key. Add it to the kube-state-metrics `--metric-labels-allowlist`, then re-run the [Step 2 verification query](#step-2-verify-the-label-is-visible-to-opencost).
- The window predates Prometheus retention, so no samples exist. Narrow the `window`.

When a PR genuinely has no data, the Action posts a comment saying so and **does not fail the build** — the signal degrades gracefully.

### Network cost is always `0`

Expected unless the OpenCost **egress DaemonSet** is enabled. Without it, egress is not measured and shows as `0`. This is a documented lower-bound limitation, not a misconfiguration.

### OpenCost is unreachable / the Action fails hard

The runner cannot reach the OpenCost API on port `9003`.

- Confirm the runner satisfies the [reachability prerequisite](#2-the-opencost-api-is-reachable-from-where-the-workflow-runs) — for hosted runners, OpenCost must be exposed via ingress/tunnel and `opencost-url` set to that address.
- Test from a pod or host on the same network: `curl -s http://opencost.opencost.svc.cluster.local:9003/allocation?window=1h`.

An unreachable OpenCost (or bad configuration) is a hard failure: the Action exits non-zero.

### Wrong port (`9003` vs `9090`)

`opencost-url` must point at the **API on `9003`**. Port `9090` is the OpenCost UI, not the allocation API — pointing ephemeractl at `9090` will not return allocation data. Re-check `opencost-url`.
