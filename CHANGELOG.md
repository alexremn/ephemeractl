# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Released versions are also available as auto-generated notes on the
[GitHub Releases](https://github.com/alexremn/ephemeractl/releases) page.

## [Unreleased]

## [1.0.0] - 2026-06-18

### Added
- Initial release: the ephemeractl GitHub Action. On a pull request it
  queries OpenCost for the actual running cost so far of that PR's Kubernetes
  preview environment and upserts a sticky cost comment, with an optional
  per-team breakdown.
- Label or namespace selector for mapping a PR to its workloads.
- GitHub Enterprise Server support via `GITHUB_API_URL`.
- Distroless Docker action image published to `ghcr.io/alexremn/ephemeractl`.

[Unreleased]: https://github.com/alexremn/ephemeractl/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/alexremn/ephemeractl/releases/tag/v1.0.0
