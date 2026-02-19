# Contribution Guide

**Generated:** 2026-02-01  
**Part:** main

Summary of contribution guidelines from [CONTRIBUTING.md](../CONTRIBUTING.md).

## Code of Conduct

The project has a [Code of Conduct](../CODE_OF_CONDUCT.md). Contributors should feel comfortable in discussion; CoC applies throughout.

## Before You Start

- **Small changes:** Fine to open an issue or PR directly.
- **More than a few lines:** Talk to someone first (Discord or GitHub issues) so your work aligns with direction and isn’t rejected after the fact.
- **Larger or architectural changes:** Discuss before implementing; roadmap and impact may not be obvious from docs.

**Preferred communication:** Discord or GitHub issues.

## Pull Request Guidelines

- **One logical change per PR.** Keep PRs small and focused. Large, multi-purpose PRs are hard to review and may be rejected; break big work into smaller PRs.
- **Use the template:** See [.github/PULL_REQUEST_TEMPLATE.md](../.github/PULL_REQUEST_TEMPLATE.md) when submitting a PR.

## Mechanics

- **Lint:** Run `make codestyle` before submitting (format + golangci-lint).
- **Tests:** CI runs `go test ./...`; ensure tests pass locally.
- **Translations:** If you change user-facing strings, update `po/default.pot` and run `make mo` as needed.

## Where to Get Help

- Not sure where to start: open an issue.
- Questions or ideas: Discord or GitHub issues.
