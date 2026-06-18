# Security Policy

ephemeractl is a self-hostable GitHub Action: it reads PR context, queries an
in-cluster OpenCost API, and posts a sticky cost comment on the PR. This document
covers supported versions, how to report a vulnerability privately, and the
project's security posture.

## Supported versions

ephemeractl is distributed as a versioned Docker action (`alexremn/ephemeractl@v1`,
image `ghcr.io/alexremn/ephemeractl`). Security fixes land on the latest minor of
the current major and are published as a new patch tag; the `v1` moving tag is
advanced to point at it.

| Version | Supported |
|---------|-----------|
| `v1.x` (latest minor) | Yes |
| Older `v1` minors | Upgrade to the latest `v1.x` |
| Pre-release / `main` | No |

Pin to a published tag (`@v1` or an immutable `@v1.2.3`) rather than `@main`.

## Reporting a vulnerability

**Report privately. Do not open a public issue, discussion, or PR for a suspected
vulnerability** — that discloses it before a fix is available.

Use either channel:

- **GitHub Security Advisories** (preferred) — open a private report via the
  repository's **Security → Report a vulnerability** tab. This keeps the discussion
  private and lets us coordinate a fix and advisory in one place.
- **Email** — `alexander.remniov@gmail.com`. If you want to encrypt, say so in a
  first plaintext message and we will exchange keys.

Helpful details to include: affected version/tag, a description of the issue and its
impact, reproduction steps or a proof of concept, and any suggested remediation.
Please do not include live secrets (tokens, kubeconfigs) in the report.

### Disclosure timeline

- **Acknowledgement:** within **72 hours** of your report.
- **Triage & severity assessment:** within **7 days**, with an initial assessment
  shared with you.
- **Fix target:** for confirmed issues, a patched release within **90 days** of
  acknowledgement; sooner for high-severity issues, and as fast as practical for any
  actively exploited vulnerability.
- **Disclosure:** coordinated. We publish a GitHub Security Advisory (and credit you,
  if you wish) once a fix is available. Please give us a reasonable window to ship
  before any public disclosure.

This is a small open-source project maintained on a best-effort basis; these are
targets, not contractual SLAs. We will keep you updated if a fix needs longer.

## Security posture

### Least-privilege GitHub token

The Action only needs to write the sticky PR comment, so the consuming workflow
should grant exactly that and nothing more:

```yaml
permissions:
  pull-requests: write
```

Do not grant broader scopes (`contents: write`, `id-token: write`, etc.) unless your
own workflow needs them for other steps. The default `github-token` input is
`${{ github.token }}` — the job-scoped `GITHUB_TOKEN` — which is preferable to a
long-lived personal access token. If you must supply a custom token, scope it to the
minimum required and store it as an encrypted repository/organization secret.

### Fork PRs are read-only by design

On `pull_request` events triggered **from a fork**, `GITHUB_TOKEN` is read-only and
has no access to secrets. The Action therefore **cannot post or update a comment from
a fork PR** — this is a GitHub security boundary, not a bug, and ephemeractl does not
attempt to work around it.

Do **not** "fix" this by switching to `pull_request_target` and checking out
untrusted PR head code: that runs fork-authored code with a privileged token and a
populated secrets context, and is a well-known privilege-escalation foot-gun. The
secure pattern, documented in `docs/USAGE.md`, is the two-workflow `workflow_run`
approach: an untrusted workflow runs on the PR, and a separate trusted workflow posts
the comment with the elevated token. (v1 documents this path; it is not implemented
in v1 code.)

### OpenCost API has no auth by default

The OpenCost allocation API (`…:9003/allocation`) ships with **no authentication**.
Anyone who can reach it can read your cluster's cost and allocation data. ephemeractl
talks to it over plain HTTP using the default in-cluster Service URL
`http://opencost.opencost.svc.cluster.local:9003`.

Protect it accordingly:

- **Never expose the OpenCost API to the public internet.** Keep it in-cluster.
- The default URL resolves only from inside the cluster. The intended setup is a
  **self-hosted runner in or near the cluster** so the Action can reach OpenCost
  without exposing it.
- If a hosted runner must reach OpenCost, front it with an authenticated, TLS-
  terminated ingress or a private tunnel — not a bare public endpoint. Restrict
  access with a **`NetworkPolicy`** (or equivalent) so only the runner can reach the
  API, and add ingress-level authentication.
- Treat cost/allocation data as sensitive; it can leak namespace names, team labels,
  and infrastructure shape.

### Token and secret handling

- The Action does **not** log the GitHub token or any other secret. Treat any
  appearance of a token in logs as a vulnerability and report it.
- It does not persist tokens to disk and does not transmit them anywhere other than
  the GitHub API.
- It does not require, read, or store cluster credentials beyond the configured
  OpenCost URL.

### Pinning and supply chain

For stronger guarantees, pin the action and image to an immutable reference (a
specific `vX.Y.Z` tag or a digest) rather than a moving major tag. Released images
are published to `ghcr.io/alexremn/ephemeractl`.

## Scope

ephemeractl v1 is the GitHub Action only. Reports about the Action's handling of
tokens, secrets, PR comments, OpenCost queries, inputs, and its published image are
in scope. Vulnerabilities in OpenCost, Kubernetes, GitHub Actions itself, or your
own workflow/cluster configuration are out of scope here and should be reported to
those projects.
