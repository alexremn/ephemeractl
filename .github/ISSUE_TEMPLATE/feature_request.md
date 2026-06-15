---
name: Feature request
about: Propose a change or addition to the ephemeractl Action
title: "[feat] "
labels: enhancement
---

<!--
ephemeractl is deliberately a thin slice: v1 is the GitHub Action only — query OpenCost,
render a sticky cost comment, keep it updated. The validate-first roadmap (docs/ROADMAP.md)
gates everything heavier (richer per-team rollup/trend, scale-to-zero, controller/CRD,
multi-cluster dashboard, budgets/chargeback) on the prior step proving adoption.

Please read docs/ROADMAP.md before filing. Requests that ask v1 to absorb a later,
adoption-gated stage will usually be deferred rather than rejected — say where it fits.
-->

## Problem / motivation

What problem are you hitting? Who is affected, and how often? Describe the situation, not a
solution. (e.g. "I can't tell which team owns the spend on a shared preview namespace.")

## Proposed solution

What you'd like ephemeractl to do. If it touches the Action interface, name the concrete
input/output or comment change (e.g. a new input, a new column in the breakdown table, a
different selector behavior).

## Alternatives considered

Other approaches you tried or rejected — existing inputs (`team-label`, `namespace-pattern`,
`window`, `idle-mode`), a different OpenCost query, an upstream OpenCost/Kubernetes change,
or solving it outside ephemeractl (kube-janitor, ArgoCD, your own workflow step).

## How it fits the roadmap (validate-first)

Where does this sit relative to docs/ROADMAP.md?

- [ ] Improves the **v1 Action** itself (cost query, rendering, selector, comment behavior)
- [ ] Belongs to a **later, adoption-gated stage** (per-team trend, scale-to-zero, controller/CRD, dashboard, budgets) — filing to track interest
- [ ] Explicitly **out of scope** today, but worth recording

Note: per the validate-first model, even strong ideas for later stages stay parked until the
current thin slice proves adoption. Stating which stage this is keeps prioritization honest.

## Additional context

Anything else — links, similar tools (Kubecost `cost-prediction-action`, Infracost), or the
specific OpenCost / cluster setup that motivates this.
