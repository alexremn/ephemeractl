# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Released versions are also available as auto-generated notes on the
[GitHub Releases](https://github.com/alexremn/ephemeractl/releases) page.

## [Unreleased]

### Changed
- Build the action image on `golang:1.26-alpine`.
- Bump pinned CI/release actions to their latest majors (Node 24 runtimes):
  checkout v7, docker login v4, buildx v4, build-push v7, action-gh-release v3.
- Release notes are now sourced from the matching `CHANGELOG.md` section, with
  the auto-generated commit list appended below.

## [1.0.0] - 2026-06-18

### Added
- Initial release: the ephemeractl GitHub Action. On a pull request it
  queries OpenCost for the actual running cost so far of that PR's Kubernetes
  preview environment and upserts a sticky cost comment, with an optional
  per-team breakdown.
- Label or namespace selector for mapping a PR to its workloads.
- GitHub Enterprise Server support via `GITHUB_API_URL`.
- Distroless Docker action image published to `ghcr.io/alexremn/ephemeractl`.

### Security
- All CI/release GitHub Actions pinned to full commit SHAs; Dependabot enabled
  to keep them current.
- Distroless nonroot runtime image; multi-arch (amd64/arm64) with SLSA build
  provenance and SBOM attestations published on each release.

[Unreleased]: https://github.com/alexremn/ephemeractl/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/alexremn/ephemeractl/releases/tag/v1.0.0
