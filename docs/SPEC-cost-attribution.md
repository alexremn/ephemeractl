# Cost attribution spec

> How ephemeractl decides which workloads belong to a PR, how it asks OpenCost for
> their cost, how it adds that cost up, and exactly how far you can trust the number.
>
> This is the document to read before you put the figure in front of an engineer or a
> FinOps owner. It is deliberately precise about what the number *is* and what it is
> *not*. ephemeractl reports an **approximate lower bound on actual running spend** — a
> trustworthy relative signal and trend, not an invoice you can reconcile.

ephemeractl: actual running cost of every PR's preview environment, posted on the PR.

---

## 1. The selector — finding "this PR's" workloads

A PR's preview environment is some set of Kubernetes workloads. ephemeractl has to
translate "PR #123" into a filter OpenCost understands. There is no universal industry
convention for how preview envs are tagged, so the Action exposes **one selector**,
resolved from whichever input is set. Both modes compile down to a single OpenCost
filter parameter.

### 1.1 Label selector (default, recommended)

Workloads carry a label whose **value is the PR number**. The key is configurable via
the `pr-label-key` input; the default is `ephemeractl.dev/pr`.

```yaml
# spec.template.metadata.labels — NOT just the Deployment's own labels
ephemeractl.dev/pr: "123"
```

This compiles to:

```
&filterLabels=ephemeractl.dev/pr:123
```

**The label MUST sit on the pod template** (`spec.template.metadata.labels`), not only
on the Deployment / ReplicaSet metadata. OpenCost attributes cost at the **pod** level,
because cost is derived from pod resource usage scraped by Prometheus via kube-state-metrics
(KSM). A label that exists on the Deployment but is not propagated to the pods it creates
is invisible to the allocation query — those pods land in OpenCost's `__unallocated__`
bucket and ephemeractl reports `0` for the PR. This is the single most common reason for a
"why is my PR showing no cost?" report. See [§7.1](#71-prometheusksm-label-sanitization)
for how the key itself is transformed before you can match on it.

Why the label is the default:

- **Immutable across the PR's life.** The PR number does not change; the label does not
  need to be rewritten as the env evolves.
- **Robust across multiple namespaces.** A PR env that spans several namespaces (app +
  dependencies + jobs) is still one label value, so it sums correctly.
- **Survives namespace reuse.** Combined with a time-bounded window ([§4](#4-window-resolution)),
  it does not conflate the current PR with a previous tenant of the same namespace.

### 1.2 Namespace selector (alternative)

If you set `namespace-pattern`, ephemeractl maps PR → namespace by substituting the PR
number into the pattern. For example `preview-pr-{pr}` with PR #123 resolves to the
namespace `preview-pr-123`, which compiles to:

```
&filterNamespaces=preview-pr-123
```

Setting `namespace-pattern` **overrides** label mode. This matches the dominant
one-namespace-per-PR convention and needs no pod-template labeling. Its limitations:

- **Breaks on multi-namespace envs.** Only the matched namespace is counted; cost in
  sibling namespaces is silently dropped.
- **Vulnerable to namespace reuse.** If `preview-pr-123` is torn down and the name is
  later recycled, an un-bounded window mixes both tenants' cost. Always time-bound the
  window ([§4](#4-window-resolution)); prefer the label selector if reuse is possible.

Because of these failure modes the namespace selector is the fallback, not the default.

### 1.3 Exactly one selector is sent

The Action sends **exactly one** of `filterLabels` or `filterNamespaces` per query —
never both. Resolution order:

1. `namespace-pattern` set → namespace mode (`filterNamespaces`).
2. Otherwise → label mode (`filterLabels`) using `pr-label-key`.

---

## 2. The OpenCost query

ephemeractl issues a single HTTP `GET` against the OpenCost Allocation API. The default
base URL is the in-cluster Service:

```
http://opencost.opencost.svc.cluster.local:9003
```

Port **9003** is the OpenCost API. It is **not** 9090 (that is the OpenCost UI, a different
service). OpenCost has **no authentication by default**, so no credentials are sent. The
runner must be able to reach this URL — typically a self-hosted runner in or near the
cluster (see USAGE.md for the network prerequisite).

### 2.1 The request

```
GET {opencost-url}/allocation
  ?window={resolved-window}              # see §4
  &accumulate=true                       # collapse the whole window into ONE set
  &resolution={opencost-resolution}      # default 1m

  # selector — exactly one of (see §1):
  &filterLabels={pr-label-key}:{pr}      # label mode (default)
  &filterNamespaces={resolved-namespace} # namespace mode

  # aggregation:
  &aggregate=label:{team-label}          # when team-label is set → per-team groups
  # (no team-label → aggregate=namespace, one rolled-up figure)

  # idle policy (from idle-mode, see §5):
  &includeIdle={true|false}              # include-idle → true,  used-only → false
  &shareIdle={true|false}                # include-idle → true,  used-only → false
```

### 2.2 Parameter reference

| Param | Source | Value | Why |
|-------|--------|-------|-----|
| `window` | `window` input | resolved per [§4](#4-window-resolution) | The time range to bill. `pr-open` → `created_at,now`. |
| `accumulate` | fixed | `true` | Collapses every per-resolution slice in the window into a single accumulated allocation set, so one PR yields one number (per group) instead of a time series. |
| `resolution` | `opencost-resolution` input | `1m` (default) | Sampling granularity OpenCost uses when computing allocations. Finer = more accurate for short-lived pods, more load on OpenCost/Prometheus. |
| `filterLabels` | label selector | `{pr-label-key}:{pr}` | Restricts to pods carrying the PR label. |
| `filterNamespaces` | namespace selector | `{resolved-namespace}` | Restricts to the PR's namespace. |
| `aggregate` | `team-label` input | `label:{team-label}` if set, else `namespace` | Controls the grouping of returned allocations: one group per team, or one rolled-up figure. |
| `includeIdle` | `idle-mode` input | `true` for `include-idle`, else `false` | Whether unused-but-reserved capacity in scope is surfaced. |
| `shareIdle` | `idle-mode` input | `true` for `include-idle`, else `false` | Whether that idle cost is distributed across the returned allocations. |

### 2.3 The response shape

OpenCost returns a JSON envelope. The relevant part is `data`, an array of **allocation
sets**. With `accumulate=true` there is exactly one set. Each set is a map keyed by the
aggregation key (team name, namespace, or `__unallocated__`), whose values are allocation
objects carrying the cost components ephemeractl sums in [§3](#3-summing-the-cost).

```jsonc
{
  "code": 200,
  "data": [
    {
      "<group-key>": {
        "name": "<group-key>",
        "cpuCost": 0.0,
        "ramCost": 0.0,
        "gpuCost": 0.0,
        "networkCost": 0.0,
        "loadBalancerCost": 0.0,
        "sharedCost": 0.0,
        "pvs": {
          "<cluster>/<pv-name>": { "cost": 0.0 }
        }
      }
    }
  ]
}
```

> A worked, realistic response is in [§8](#8-worked-example).

---

## 3. Summing the cost

This is the correctness-critical part. **OpenCost's allocation JSON has no `totalCost`
field and no flat `pvCost` field.** A consumer that reads either of those gets `0` and
reports a wrong (low) number. ephemeractl computes the cost of each allocation by summing
its components explicitly:

```
cost(allocation) =
      cpuCost
    + ramCost
    + gpuCost
    + networkCost
    + loadBalancerCost
    + sharedCost
    + Σ pvs[*].cost          # PV cost lives inside the "pvs" map, per-volume — NOT a flat field
```

- **Persistent volume cost** is keyed under `pvs`, one entry per volume, each with its own
  `cost`. You must iterate the map and sum every entry's `cost`. There is no pre-summed
  `pvCost`.
- Any component absent from the JSON is treated as `0`.

Then:

```
total      = Σ cost(allocation)  over all returned groups
per-team   = cost(group)         for each group, when aggregated by team-label
```

When `aggregate=namespace` (no `team-label`), there is typically one group and `total` is
its cost. When `aggregate=label:{team-label}`, each group is a team and the per-team table
is the per-group sums; `total` is their sum.

The `__unallocated__` group, if present, signals workloads in scope that OpenCost could not
attribute — usually a label-on-Deployment-not-pod-template mistake ([§1.1](#11-label-selector-default-recommended))
or a label the metrics pipeline dropped ([§7.1](#71-prometheusksm-label-sanitization)).
ephemeractl surfaces it rather than hiding it, because a large `__unallocated__` figure
means the PR's real cost is *higher* than the attributed number.

---

## 4. Window resolution

The `window` input decides the time range billed.

### 4.1 `pr-open` (default) — lifetime spend of the env

`pr-open` resolves to the **RFC3339 pair** `{pull_request.created_at},{now}`:

```
window=2026-06-12T09:14:03Z,2026-06-15T21:24:00Z
```

- `pull_request.created_at` comes from the GitHub event payload (`$GITHUB_EVENT_PATH`).
- `now` is the Action's wall-clock time at execution, formatted RFC3339 (UTC).

This bills the **entire life of the preview environment so far** — the headline "what has
this PR cost to date" number.

### 4.2 Native OpenCost windows

`window` also accepts any window OpenCost understands, passed through verbatim:

- Duration: `7d`, `24h`, `30m`
- Named: `today`, `yesterday`, `week`, `lastweek`, `month`, `lastmonth`
- RFC3339 pair: `2026-06-12T00:00:00Z,2026-06-15T00:00:00Z`
- Unix timestamp pair: `1749718800,1750022640`

### 4.3 Bounded by Prometheus retention

The window cannot reach further back than OpenCost's backing **Prometheus retention**,
commonly **~15 days** unless extended. A `pr-open` window for a PR older than retention
silently starts at the edge of available data — the figure then covers "since retention
began," not "since the PR opened." For long-lived PRs, extend Prometheus retention or
treat the number as a trailing-window cost. This is a property of the data source, not a
bug in ephemeractl, and it is one more reason the figure is a lower bound.

---

## 5. Idle and shared cost policy

Whether to count **idle** capacity (reserved-but-unused CPU/RAM) is a policy choice, not a
fact. ephemeractl exposes it as `idle-mode`:

| `idle-mode` | `includeIdle` | `shareIdle` | Meaning |
|-------------|---------------|-------------|---------|
| `used-only` (default) | `false` | `false` | Bill only resources the PR's pods actually used. The tightest, most defensible "what this PR consumed" figure. |
| `include-idle` | `true` | `true` | Also surface idle capacity in scope and distribute it across the PR's allocations. Closer to "share of the cluster the PR reserved." |

The default is **`used-only`**, and this is a deliberate, honest limitation. Bare OpenCost
— unlike Kubecost — has **no `shareNamespaces` / `shareCost` knobs** for proportionally
redistributing shared overhead. So ephemeractl does not pretend to do sophisticated shared-cost
allocation: it reports used cost by default and, at most, OpenCost's own idle handling when
you opt in. Anything beyond that would be inventing precision the data source does not
provide.

---

## 6. The honesty model

The number is an **approximate lower bound**. Every PR comment says so. These are the
specific reasons it is a floor, not an exact figure:

- **On-demand list rates only.** OpenCost prices resources at public on-demand list rates.
  It does **not** reconcile spot/preemptible discounts, reserved instances, savings plans,
  or committed-use discounts. Your real negotiated/discounted cost is typically **lower**;
  the reported figure is an upper bound on per-unit price but a lower bound on coverage —
  see the next points.
- **Network egress is `0` by default.** `networkCost` is `0` unless OpenCost's network-cost
  egress DaemonSet is enabled. Real egress cost is therefore usually under-counted.
- **Leaked / unmounted PVs and some load-balancer cost may be undercounted.** Volumes not
  currently mounted to a running pod, and some LB cost, live in OpenCost **Assets**, not the
  **Allocation** API ephemeractl queries — so they can be missed.
- **GPU needs explicit pricing config.** `gpuCost` is only populated when GPU pricing is
  configured in OpenCost; otherwise GPU spend reads as `0`.
- **Short-lived pods can be under-sampled.** At `1m` resolution, pods that live for seconds
  may fall between samples and be under-counted. Lower `opencost-resolution` to tighten this
  at the cost of more query load.

The honest framing, stated plainly in the comment and the README:

> Trustworthy **relative signal and trend** — compare PRs, watch a PR's cost climb — **not
> invoice reconciliation.** Treat it as a floor, not a final bill.

This honesty is the point. It is what earns the number a place in front of engineers and
FinOps owners.

---

## 7. Sharp edges

Document these; do not hide them. Each one can make the number wrong if ignored.

### 7.1 Prometheus / KSM label sanitization

Kubernetes label keys are **sanitized** before they reach Prometheus via
kube-state-metrics. A key like `ephemeractl.dev/pr` becomes a metric label
`label_ephemeractl_dev_pr` — dots and slashes are replaced with underscores and a `label_`
prefix is added. Additionally, **KSM may not export your custom label at all** unless it is
allow-listed: you may need to start kube-state-metrics with

```
--metric-labels-allowlist=pods=[ephemeractl.dev/pr]
```

(or the equivalent for your KSM deployment). If the label is not exported, OpenCost never
sees it, and every matched pod falls into `__unallocated__` — the query returns `0` for the
PR.

**Verify before trusting numbers.** Run one sample aggregate query against your OpenCost
and confirm your label appears as a group key:

```bash
curl -s "http://opencost.opencost.svc.cluster.local:9003/allocation\
?window=24h&aggregate=label:ephemeractl.dev/pr&accumulate=true" \
  | jq '.data[0] | keys'
```

If you see your PR values (e.g. `"123"`) as keys, the pipeline is wired correctly. If you
see only `"__unallocated__"`, the label is not on the pod template, not exported by KSM, or
named differently than configured. This one-line check is in USAGE.md as a required setup
step.

### 7.2 Namespace reuse / recycled PR numbers

PR numbers and namespace names get recycled. An un-bounded window over a recycled namespace
mixes the cost of the current PR with a previous tenant. Two defenses, both used by default:

- **Always time-bound the window.** `pr-open` starts at *this* PR's `created_at`, so cost
  before the PR existed is excluded.
- **Prefer the immutable label selector.** The `ephemeractl.dev/pr` value is tied to the PR
  number, not to a reusable namespace name, so it does not conflate distinct PRs sharing a
  namespace name over time.

### 7.3 Fork-PR token model

On a `pull_request` event from a **fork**, GitHub gives the workflow a **read-only
`GITHUB_TOKEN` with no access to secrets**. The Action therefore **cannot post or update the
PR comment** for fork PRs. This is a GitHub platform constraint, not an ephemeractl bug.

v1 documents this as a known limitation. The secure upgrade path is the **`workflow_run`
two-workflow pattern**: an untrusted `pull_request` workflow gathers context, then a trusted
`workflow_run` workflow (which has write permissions and secret access) does the commenting.
**This is not solved in v1 code** — it is a documented limitation with a documented path
forward, nothing more.

---

## 8. Worked example

A concrete end-to-end run, so the math is unambiguous. (Numbers here are an independent
illustration — a different PR and scenario from the sample comment in the README.)

**Scenario:** PR **#123** opened a preview environment. Its pods are labeled
`ephemeractl.dev/pr: "123"` on the pod template. No `team-label` is set, so aggregation is
by namespace and we want one total. Window is the default `pr-open`. Idle mode is the
default `used-only`.

### 8.1 Resolved window

The PR was created at `2026-06-12T09:14:03Z`; the Action runs at `2026-06-15T21:24:00Z`:

```
window=2026-06-12T09:14:03Z,2026-06-15T21:24:00Z
```

### 8.2 The request

```bash
curl -s "http://opencost.opencost.svc.cluster.local:9003/allocation\
?window=2026-06-12T09:14:03Z,2026-06-15T21:24:00Z\
&accumulate=true\
&resolution=1m\
&filterLabels=ephemeractl.dev/pr:123\
&aggregate=namespace\
&includeIdle=false\
&shareIdle=false"
```

### 8.3 A representative response

```jsonc
{
  "code": 200,
  "data": [
    {
      "preview-pr-123": {
        "name": "preview-pr-123",
        "cpuCost": 1.842,
        "ramCost": 0.913,
        "gpuCost": 0.0,
        "networkCost": 0.0,          // egress DaemonSet not enabled → 0 (see §6)
        "loadBalancerCost": 0.220,
        "sharedCost": 0.0,           // used-only mode
        "pvs": {
          "cluster-1/pvc-app-data": { "cost": 0.310 },
          "cluster-1/pvc-postgres": { "cost": 0.475 }
        }
      }
    }
  ]
}
```

### 8.4 The computed total

Apply the formula from [§3](#3-summing-the-cost) to the single group `preview-pr-123`:

```
cpuCost            =  1.842
ramCost            =  0.913
gpuCost            =  0.000
networkCost        =  0.000
loadBalancerCost   =  0.220
sharedCost         =  0.000
Σ pvs[*].cost      =  0.310 + 0.475  =  0.785
                      -------------------------
cost(group)        =  3.760
```

There is one group, so:

```
total = 3.760
```

ephemeractl reports `total-cost = 3.76`, formats it with the `currency` input (`USD`), and
upserts the sticky PR comment (first line `<!-- ephemeractl:cost-report -->`) with the
total, the breakdown, the resolved window + idle line, and the honesty note. The figure is
presented as an approximate lower bound — note the `networkCost: 0` above is exactly the
kind of undercount the honesty model warns about.

---

## See also

- `docs/USAGE.md` — prerequisites, the verify-your-label step, the consuming workflow.
- `docs/ARCHITECTURE.md` — the components that build this query and render the comment.
- `action.yml` — the canonical input/output contract referenced throughout this spec.
