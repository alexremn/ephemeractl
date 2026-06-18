---
name: Bug report
about: Report incorrect cost figures, comment failures, or other defects in the ephemeractl Action
title: "[bug] "
labels: bug
---

<!--
Thanks for filing a bug. ephemeractl reports an approximate lower-bound cost from
OpenCost, so before reporting a "wrong number", please check docs/SPEC-cost-attribution.md
to confirm the figure is actually wrong and not an expected limitation (no spot/RI rates,
egress 0 without the DaemonSet, undercounted PV/LB cost, etc.).

Fill in every section. Issues missing the environment or the OpenCost reachability check
are much slower to triage.
-->

## Environment

- **ephemeractl version / Action tag** (e.g. `alexremn/ephemeractl@v1` or `@v1.2.3` or pinned SHA):
- **Kubernetes version** (`kubectl version --short`):
- **OpenCost version** (chart/image tag):
- **Runner type**: self-hosted / GitHub-hosted
- **Selector mode**: label (`pr-label-key`) / namespace (`namespace-pattern`)
  - Selector value in use (the `pr-label-key` or `namespace-pattern` you configured):

## What happened

A clear description of the actual behavior.

## What you expected

A clear description of the expected behavior.

## Rendered comment output

Paste the full body of the sticky PR comment ephemeractl posted (the block that starts
with `<!-- ephemeractl:cost-report -->`). If no comment was posted, say so and paste the
relevant Action step logs instead.

```
<!-- paste the rendered comment here -->
```

## OpenCost reachability check

The Action runs on a GitHub runner and the OpenCost API must be reachable from that runner.
Run the equivalent check **from the runner** (or a pod in the same network) against the
`opencost-url` you configured, and paste the result:

```bash
# default in-cluster URL shown; substitute your opencost-url
curl -sS "http://opencost.opencost.svc.cluster.local:9003/allocation?window=today&accumulate=true" | head -c 2000
```

```
<!-- paste the curl output (or the error) here -->
```

## Action configuration

The relevant `with:` block from your workflow (redact tokens):

```yaml
- uses: alexremn/ephemeractl@v1
  with:
    opencost-url: ...
    pr-label-key: ...
    # window, team-label, idle-mode, namespace-pattern, etc.
```

## Additional context

Anything else that helps — multi-namespace setup, recycled PR numbers, KSM label allowlist
(`--metric-labels-allowlist`), custom window, fork-PR scenario, etc.
