---
title: kcal-cli Documentation
---

# kcal-cli Documentation

Purpose: Single-page documentation for `kcal`, optimized for browser find/search on GitHub Pages.

## Sections

- [Installation](#installation)
- [Getting Started](#getting-started)
- [Command Reference](#command-reference)
- [Config and Paths](#config-and-paths)
- [Barcode Providers](#barcode-providers)
- [Lookup Overrides and Cache](#lookup-overrides-and-cache)
- [Import and Export](#import-and-export)
- [Backups and Recovery](#backups-and-recovery)
- [Analytics and Goals](#analytics-and-goals)
- [Development](#development)
- [Releases](#releases)

> Looking for the repository quick start? See the root `README.md`.

---


## Installation

Purpose: Install `kcal` locally for day-to-day CLI usage.

When to use this page:
- You are installing `kcal` for the first time.
- You are choosing between source install and Homebrew.

## Option 1: Homebrew

```bash
brew tap saadjs/kcal
brew install kcal
```

## Option 2: Go Install

```bash
go install github.com/saad/kcal-cli@latest
kcal --help
```

## Option 3: Build from Source

```bash
git clone https://github.com/saad/kcal-cli.git
cd kcal-cli
go build -o kcal .
./kcal --help
```

## Next Step

- Continue to [Getting Started](#getting-started).

## Failure and Edge Cases

- If `go install` succeeds but `kcal` is not found, add your Go bin directory to `PATH`.
- If local build fails, run `go mod download` and retry.
- If Homebrew install fails, verify tap access and run `brew update` before retrying.

---


## Getting Started

Purpose: Run a minimal end-to-end flow for logging food and reviewing daily progress.

When to use this page:
- You want to be productive with `kcal` in under 5 minutes.
- You are setting up a fresh local database.

## Minimal Happy Path

```bash
kcal init
kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-20
kcal entry add --name "Chicken bowl" --calories 550 --protein 45 --carbs 40 --fat 18 --category lunch
kcal today
kcal analytics week
```

## Common Next Commands

```bash
kcal category add snacks
kcal entry quick "Greek Yogurt | 220 20 18 8 | breakfast"
kcal goal current
kcal body add --weight 172 --unit lb --date 2026-02-20 --time 07:00
```

## Where to Go Next

- [Command Reference](#command-reference)
- [Analytics and Goals](#analytics-and-goals)
- [Barcode Providers](#barcode-providers)

## Failure and Edge Cases

- `kcal goal set` requires full macro inputs; verify all required flags are provided.
- Invalid date formats must be `YYYY-MM-DD`.
- If you use custom DB locations, append `--db /path/to/kcal.db`.

---


## Command Reference

Purpose: Provide a complete command map without repeating every `--help` flag listing.

When to use this page:
- You want to discover command families and subcommands quickly.
- You need copy-paste examples per domain.

## Top-Level Commands

- `analytics`
- `backup`
- `body`
- `body-goal`
- `category`
- `completion`
- `config`
- `doctor`
- `entry`
- `exercise`
- `export`
- `goal`
- `import`
- `init`
- `lookup`
- `recipe`
- `today`

## Nutrition Logging

- `kcal category add|list|rename|delete`
- `kcal entry add|quick|list|show|update|metadata|delete|search|repeat`

Example:

```bash
kcal entry quick "Oats | 300 12 45 8 | breakfast" --date 2026-02-20 --time 08:00
kcal entry search --query oats --limit 10
kcal entry repeat 12 --date 2026-02-21 --time 08:00
```

## Goals and Body

- `kcal goal set|current|history|suggest`
- `kcal body add|list|update|delete`
- `kcal body-goal set|current|history`

Example:

```bash
kcal goal suggest --weight 80 --unit kg --maintenance-calories 2500 --pace cut --apply --effective-date 2026-02-20
kcal body-goal set --target-weight 170 --unit lb --target-body-fat 18 --effective-date 2026-02-20
```

## Recipes and Exercise

- `kcal recipe add|list|show|update|delete|log|recalc`
- `kcal recipe ingredient add|list|update|delete`
- `kcal exercise add|list|update|delete`

Example:

```bash
kcal recipe add --name "Overnight oats" --calories 0 --protein 0 --carbs 0 --fat 0 --servings 2
kcal recipe ingredient add "Overnight oats" --name Oats --amount 40 --unit g --calories 150 --protein 5 --carbs 27 --fat 3
kcal recipe recalc "Overnight oats"
kcal recipe log "Overnight oats" --servings 1 --category breakfast
```

## Analytics

- `kcal analytics week|month|range`
- `kcal analytics insights week|month|range`

Example:

```bash
kcal analytics month --month 2026-02
kcal analytics range --from 2026-02-01 --to 2026-02-20
kcal analytics insights range --from 2026-02-01 --to 2026-02-20 --granularity auto --out insights.md --out-format markdown
```

## Data and Integrity

- `kcal backup create|list|restore`
- `kcal export --format json|csv --out <file>`
- `kcal import --format json|csv --in <file> [--mode fail|skip|merge|replace] [--dry-run]`
- `kcal doctor [--fix]`

Example:

```bash
kcal export --format json --out backup.json
kcal import --format json --in backup.json --mode merge --dry-run
kcal doctor --fix
```

## Lookup and Providers

- `kcal lookup barcode <code>`
- `kcal lookup search --query <text> [--fallback --limit N --verified-only --verified-min-score X]`
- `kcal lookup providers`
- `kcal lookup usda-help|openfoodfacts-help|upcitemdb-help`
- `kcal lookup override set|show|list|delete`
- `kcal lookup cache list|purge|refresh`
- `kcal lookup cache search-list|search-purge`

Example:

```bash
kcal lookup barcode 3017620422003 --provider openfoodfacts
kcal lookup search --query "greek yogurt" --fallback --limit 10 --verified-only
kcal lookup override list --provider openfoodfacts
kcal lookup cache purge --provider openfoodfacts --all
kcal lookup cache search-list --provider openfoodfacts --query "greek yogurt"
```

## Command Help

For flag-level details, use:

```bash
kcal <command> --help
```

## Failure and Edge Cases

- Subcommands that mutate by ID (for example `entry update <id>`) fail if the ID does not exist.
- Some commands require required flags and return explicit errors (for example `import --in`, `backup restore --file`).
- Provider lookups can fail due to missing API keys or external API limits.

---


## Config and Paths

Purpose: Explain where `kcal` stores local state and how runtime configuration is resolved.

When to use this page:
- You need to customize DB location.
- You are troubleshooting provider configuration and env variable precedence.

## Database Path

- Global flag: `--db <path>` overrides the default DB path.
- Initialize storage with `kcal init` before other commands.

Example:

```bash
kcal --db /tmp/kcal.db init
kcal --db /tmp/kcal.db today
```

## Configuration Commands

- `kcal config set --barcode-provider <provider>`
- `kcal config set --fallback-order <comma-separated-providers>`
- `kcal config set --api-key-hint <text>`
- `kcal config get`

Example:

```bash
kcal config set --barcode-provider openfoodfacts
kcal config set --fallback-order openfoodfacts,usda,upcitemdb
kcal config get
```

## Lookup Environment Variables

- `KCAL_USDA_API_KEY`
- `KCAL_BARCODE_API_KEY` (legacy fallback)
- `KCAL_UPCITEMDB_API_KEY`
- `KCAL_UPCITEMDB_KEY_TYPE`
- `KCAL_BARCODE_PROVIDER`
- `KCAL_BARCODE_FALLBACK_ORDER`

## Provider Resolution Notes

- Provider can come from `--provider`, config, or env var defaults.
- API keys can come from command flags or env vars.
- Local override and cache layers can satisfy lookup before live provider calls.

See also:
- [Barcode Providers](#barcode-providers)
- [Lookup Overrides and Cache](#lookup-overrides-and-cache)

## Failure and Edge Cases

- Invalid fallback-order values cause lookup validation failures.
- Missing provider key material causes provider-specific errors.
- Mismatched DB paths can make config appear missing.

---


## Barcode Providers

Purpose: Configure and use external nutrition lookup providers with the `kcal lookup` command family.

When to use this page:
- You want barcode-based entry enrichment.
- You need to choose provider defaults and fallback strategy.

## Supported Providers

- `usda` (default)
- `openfoodfacts`
- `upcitemdb`

Quick provider summary:

```bash
kcal lookup providers
```

## Basic Lookup Examples

```bash
kcal lookup barcode 012345678905 --provider usda
kcal lookup barcode 3017620422003 --provider openfoodfacts
kcal lookup barcode 012993441012 --provider upcitemdb
```

## USDA Setup

```bash
export KCAL_USDA_API_KEY=your_key_here
kcal lookup usda-help
kcal lookup barcode 786012004549 --provider usda
```

## Open Food Facts Setup

```bash
export KCAL_BARCODE_PROVIDER=openfoodfacts
kcal lookup openfoodfacts-help
kcal lookup barcode 3017620422003
```

## UPCitemdb Setup

```bash
export KCAL_UPCITEMDB_API_KEY=your_key_here
export KCAL_UPCITEMDB_KEY_TYPE=3scale
kcal lookup upcitemdb-help
kcal lookup barcode 012993441012 --provider upcitemdb
```

## Fallback Order

```bash
export KCAL_BARCODE_FALLBACK_ORDER="usda,openfoodfacts,upcitemdb"
kcal lookup barcode 012345678905 --fallback
```

## Text Search and Verification

```bash
kcal lookup search --query "greek yogurt" --fallback --limit 10
kcal lookup search --query "greek yogurt" --fallback --verified-only --verified-min-score 0.85
```

Notes:
- Search responses include `confidence_score` and `is_verified`.
- `--verified-min-score` controls the verification threshold used by `--verified-only`.

## See Also

- [Lookup Overrides and Cache](#lookup-overrides-and-cache)
- [Config and Paths](#config-and-paths)

## Failure and Edge Cases

- USDA lookups fail without a key unless another provider is used.
- Provider APIs may throttle requests; use cache and fallback to improve reliability.
- Invalid barcode formats can return no-match responses even with valid provider configuration.

---


## Lookup Overrides and Cache

Purpose: Control local corrections and caching behavior for barcode lookups.

When to use this page:
- Provider nutrition data is incomplete or incorrect.
- You want deterministic local lookup results.

## Override Workflow

Set a local override:

```bash
kcal lookup override set 3017620422003 \
  --provider openfoodfacts \
  --name "Nutella Custom" \
  --brand Ferrero \
  --serving-amount 15 \
  --serving-unit g \
  --calories 99 \
  --protein 1 \
  --carbs 10 \
  --fat 6 \
  --fiber 0.5 \
  --sugar 10 \
  --sodium 15
```

Inspect and list overrides:

```bash
kcal lookup override show 3017620422003 --provider openfoodfacts
kcal lookup override list --provider openfoodfacts
```

Delete an override:

```bash
kcal lookup override delete 3017620422003 --provider openfoodfacts
```

## Cache Workflow

List cache rows:

```bash
kcal lookup cache list --provider openfoodfacts --limit 50
```

List text-search cache rows:

```bash
kcal lookup cache search-list --provider openfoodfacts --query "greek yogurt" --limit 50
```

Refresh one barcode:

```bash
kcal lookup cache refresh 3017620422003 --provider openfoodfacts
```

Purge cache rows:

```bash
kcal lookup cache purge --provider openfoodfacts --all
```

Purge text-search cache rows:

```bash
kcal lookup cache search-purge --provider openfoodfacts --query "greek yogurt"
kcal lookup cache search-purge --all
```

## Resolution Order

Lookup resolution order is:
- Local override
- Local cache
- Live provider

## See Also

- [Barcode Providers](#barcode-providers)
- [Command Reference](#command-reference)

## Failure and Edge Cases

- Override commands fail if required macro fields are omitted.
- Cache refresh fails if provider auth/config is incomplete.
- Purging with narrow filters may leave rows you still expect to clear.

---


## Import and Export

Purpose: Move `kcal` data between local environments using JSON or CSV.

When to use this page:
- You are backing up a dataset for transfer.
- You are restoring data into another environment.

## Export

JSON export:

```bash
kcal export --format json --out kcal-export.json
```

CSV export:

```bash
kcal export --format csv --out kcal-export.csv
```

## Import

JSON import with merge mode:

```bash
kcal import --format json --in kcal-export.json --mode merge
```

Dry-run validation:

```bash
kcal import --format json --in kcal-export.json --mode merge --dry-run
```

CSV import:

```bash
kcal import --format csv --in kcal-export.csv
```

## JSON Import Modes

- `fail`: stop on conflicts.
- `skip`: ignore conflicting records.
- `merge`: merge compatible updates.
- `replace`: replace conflicting records.

## Safety Workflow

```bash
kcal backup create
kcal import --format json --in kcal-export.json --mode merge --dry-run
kcal import --format json --in kcal-export.json --mode merge
kcal doctor
```

## See Also

- [Backups and Recovery](#backups-and-recovery)
- [Development](#development)

## Failure and Edge Cases

- CSV imports fail when row column counts do not match supported formats.
- Invalid timestamps in CSV cause row-level parse errors.
- Imports can create category mismatches if category names are missing or malformed.

---


## Backups and Recovery

Purpose: Create recoverable database snapshots and restore safely after failure or migration mistakes.

When to use this page:
- Before large imports or destructive updates.
- During local incident recovery.

## Create Backups

Default location (alongside DB in `backups/`):

```bash
kcal backup create
```

Custom output file:

```bash
kcal backup create --out /tmp/kcal-2026-02-20.db
```

Custom backup directory:

```bash
kcal backup create --dir /tmp/kcal-backups
```

## List Backups

```bash
kcal backup list
kcal backup list --dir /tmp/kcal-backups
```

## Restore Backups

```bash
kcal backup restore --file /tmp/kcal-2026-02-20.db
kcal backup restore --file /tmp/kcal-2026-02-20.db --force
```

## Integrity Checks

```bash
kcal doctor
kcal doctor --fix
```

## Recovery Runbook

```bash
kcal backup list
kcal backup restore --file /path/to/known-good.db --force
kcal doctor
```

## See Also

- [Import and Export](#import-and-export)
- [Releases](#releases)

## Failure and Edge Cases

- Restore fails when `--file` is omitted or path is invalid.
- Restore may refuse overwrite without `--force`.
- `doctor --fix` applies only safe autofixes; unresolved issues can remain.

---


## Analytics and Goals

Purpose: Interpret trend reports, adherence, and goal progress from `kcal` analytics commands.

When to use this page:
- You need weekly, monthly, or custom-range performance views.
- You are validating goal adherence and trend direction.

## Goal Setup and Tracking

```bash
kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-01
kcal goal current
kcal goal history
```

Suggested targets:

```bash
kcal goal suggest --weight 80 --unit kg --maintenance-calories 2500 --pace cut --apply --effective-date 2026-02-20
```

## Body Tracking

```bash
kcal body add --weight 172 --unit lb --body-fat 20 --date 2026-02-20 --time 07:00
kcal body-goal set --target-weight 170 --unit lb --target-body-fat 18 --effective-date 2026-02-20
```

## Analytics Commands

```bash
kcal analytics week
kcal analytics month --month 2026-02
kcal analytics range --from 2026-02-01 --to 2026-02-20
```

Insights mode:

```bash
kcal analytics insights week
kcal analytics insights range --from 2026-02-01 --to 2026-02-20 --granularity auto --out insights.md --out-format markdown
```

## Interpretation Notes

- Standard analytics reports summarize intake, exercise, net calories, category breakdowns, and adherence.
- Insights reports include period-over-period deltas, consistency metrics, streaks, and optional chart output.
- Exercise-adjusted adherence compares against effective targets that account for logged exercise.

## See Also

- [Getting Started](#getting-started)
- [Command Reference](#command-reference)

## Failure and Edge Cases

- `analytics range` and `analytics insights range` require valid `--from` and `--to` dates.
- Sparse logging periods reduce usefulness of consistency and streak metrics.
- Missing active goals can lower adherence interpretability.

---


## Development

Purpose: Document local developer workflow and checks.

When to use this page:
- You are contributing code or docs.
- You are troubleshooting CI parity locally.

## Local Setup

```bash
git clone https://github.com/saad/kcal-cli.git
cd kcal-cli
go mod download
```

## Run Tests

```bash
go test ./...
```

## Formatting and Vet

```bash
gofmt -l $(find . -name '*.go' -not -path './vendor/*')
go vet ./...
```

## Automation Summary

This repo currently uses GitHub Actions for:
- Release publishing on version tags.
- Homebrew formula sync after release.
- GitHub Pages deployment for docs.

## Docs Maintenance

- Keep README concise and route advanced content to this single docs page.
- Validate internal docs links after edits.
- Keep examples aligned with command definitions in `cmd/kcal/*.go`.

## See Also

- [Releases](#releases)
- [Command Reference](#command-reference)

## Failure and Edge Cases

- If `go test ./...` fails at compile time, resolve branch-level API mismatches first.
- `gofmt` output listing files indicates formatting drift that should be fixed before publishing.
- In-page links should use section anchors (for example `#command-reference`) to preserve single-page navigation.

---


## Releases

Purpose: Publish tagged multi-platform binaries and keep release notes/checksums traceable.

When to use this page:
- You are preparing a versioned release.
- You need to understand automated release artifacts.

## Release Trigger

- Pushing tags matching `v*` triggers the release workflow.
- Manual dispatch is available for workflow testing.

## Build Matrix

Release workflow builds:
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

Artifacts are published with generated `checksums.txt`.

## Suggested Release Flow

```bash
go test ./...
git tag v0.1.0
git push origin v0.1.0
```

Then verify GitHub release assets and checksums.

## Checklist and History

- Track notable changes in [`CHANGELOG.md`](../CHANGELOG.md)

## Docs Drift Guard

Before tagging:
- Confirm README command map matches current top-level commands.
- Confirm docs examples still match `cmd/kcal/*.go` behavior.

## Failure and Edge Cases

- Missing tag prefix `v` prevents automatic release publish stage.
- Partial artifact uploads can happen if one matrix leg fails.
- Checksum mismatches indicate corrupted or stale release outputs.
