---
title: kcal-cli Documentation
---

# kcal-cli Documentation

Purpose: This site is the complete reference for `kcal`, a local-first calorie and macro tracking CLI.

When to use this page:
- You need a map of all guides and references.
- You want to move beyond the quick start in the repository README.

## Start Here

- [Installation](/kcal-cli/installation/)
- [Getting Started](/kcal-cli/getting-started/)
- [Command Reference](/kcal-cli/reference/commands/)
- [Config and Paths](/kcal-cli/reference/config-and-paths/)

## Guides

- [Barcode Providers](/kcal-cli/guides/barcode-providers/)
- [Lookup Overrides and Cache](/kcal-cli/guides/lookup-overrides-and-cache/)
- [Import and Export](/kcal-cli/guides/import-export/)
- [Backups and Recovery](/kcal-cli/guides/backups-and-recovery/)
- [Analytics and Goals](/kcal-cli/guides/analytics-and-goals/)

## Project Operations

- [Development](/kcal-cli/development/)
- [Releases](/kcal-cli/releases/)

## Command Examples

```bash
kcal init
kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-20
kcal today
```

## Failure and Edge Cases

- If `kcal` is not found, confirm your install path (`go install` puts binaries in your Go bin path).
- If a command fails with DB errors, initialize first with `kcal init` or pass `--db`.
- If GitHub Pages links 404 after first deploy, verify repository Pages settings are set to GitHub Actions.
