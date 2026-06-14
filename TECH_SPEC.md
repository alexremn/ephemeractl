# ephemeractl — Tech Spec

> A self-hostable Kubernetes controller (+ GitHub Action) giving per-PR preview environments hard TTLs with idempotent retrying cleanup, orphan/stale-PR scanning, sleep-on-inactivity, AND per-PR/per-team cost attribution posted back onto the PR.

## Problem

Preview envs are expected on every PR, but the cleanup + cost-attribution lifecycle is the dominant failure: orphaned envs, silent hook failures, no hard TTL, one opaque bill. The cleanup gap is the source of most preview-environment cost overruns.

## Target user

Mid-to-large orgs (100+ engineers) running self-hosted preview envs on Kubernetes, and the FinOps owners who eat the bill.

## The wedge (sharpened)

**Do not rebuild the commoditized cleanup core.** Ship ONLY the unmet slice as an add-on alongside ArgoCD/vCluster:

- **OpenCost-driven running/ACTUAL per-PR + per-team cost comment** (distinct from Kubecost's *predicted* manifest cost), and
- **scale-to-zero on idle.**

Interoperate with kube-janitor for TTL rather than reimplementing finalizers. **Ship the thin `actual-cost-on-PR` Action first** — if that one feature doesn't pull adoption, kill it before building a CRD/operator.

### Why existing tools fall short

- ArgoCD ApplicationSet PR Generator + kube-janitor — the standard free recipe; already owns per-PR create/teardown + TTL/orphan cleanup.
- Kubecost `cost-prediction-action` — posts *predicted* (manifest) cost on PRs, not actual running cost.
- Bunnyshell / Qovery / Northflank — bundle the rest but charge for it.

## Stack & form factor

- **Language:** Go
- **Libraries:** `controller-runtime`, OpenCost API, `go-github` (PR comments)
- **Form factor:** start as a thin GitHub Action, NOT a full operator. Promote to a controller only after the cost-on-PR slice proves adoption.

## Effort to v1

- **Budget more than 5–8 weeks.** The hard novel work is OpenCost-allocation → PR-identity mapping and trustworthy chargeback, plus the wrongful-deletion safety surface.

## Adoption risk

The valuable slice (cost attribution + sleep) is exactly what funded vendors monetize, so the open-core SaaS angle competes head-on. Validate with the thin "actual-cost-on-PR via OpenCost" Action ALONE before committing to a CRD/operator.

## Monetization angle

OSS controller; open-core multi-cluster cost dashboard + per-team budgets/chargeback.

## Verdict (from market scan)

need **4/5**, buildable **4/5** — **refine**. Ranked **#5 to build first**: cleanup is solved by others; win on actual-cost-on-PR + scale-to-zero, ship that slice first.
