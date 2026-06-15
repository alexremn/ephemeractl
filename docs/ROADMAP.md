# Roadmap

> **Validate first.** Nothing below the line is committed work. ephemeractl v1 is the thin cost
> Action and only the thin cost Action. Each item past step 1 stays a hypothesis until the cost
> comment proves real adoption — if teams don't pull the comment into their PR workflow, the project
> is killed before any heavier machinery is built. The ladder is adoption-gated: each rung unlocks
> only after the rung below it has earned its keep.

---

## Adoption-gated ladder

1. **Thin cost Action** _(shipping now)_ — the v1 GitHub Action: sticky PR comment with actual OpenCost spend.
2. **Richer per-team rollup & cost trend** — gated on step 1 showing sustained use across multiple teams/repos.
3. **Scale-to-zero on idle** — gated on step 2 surfacing idle preview-env spend as a real, recurring cost.
4. **Optional controller/CRD** — gated on step 3 demonstrating that in-cluster automation beats the stateless Action.
5. **Open-core multi-cluster dashboard + budgets/chargeback** — gated on step 4 proving demand for cross-cluster, org-wide cost governance.

## Monetization

OSS core (this repo, Apache-2.0) stays free and self-hostable. The commercial layer is a hosted
multi-cluster dashboard (step 5) — budgets, chargeback, and cross-cluster rollup as a managed
service. The Action and core stay open; the hosted dashboard is the paid product.

_None of steps 2–5 are specified in detail here, by design. They are named, not committed._
