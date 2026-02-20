---
nav_exclude: true
---

> Recent changes: see [`CHANGELOG.md`](https://github.com/saadjs/kcal-cli/blob/main/CHANGELOG.md).

## Quick Tasks

- Install `kcal`: [Installation](#installation)
- Log your first meal and check progress: [Getting Started](#getting-started)
- Find a command quickly: [Command Reference](#command-reference)
- Configure DB path and provider defaults: [Config and Paths](#config-and-paths)
- Set up barcode lookup providers: [Barcode Providers](#barcode-providers)
- Fix lookup results with local overrides/cache: [Lookup Overrides and Cache](#lookup-overrides-and-cache)
- Move data between machines: [Import and Export](#import-and-export)
- Restore from backup safely: [Backups and Recovery](#backups-and-recovery)
- Investigate common issues first: [Troubleshooting](#troubleshooting)

---

## Troubleshooting

Start here if commands fail unexpectedly.

- Command not found after install: make sure your Go/bin install path is in `PATH`.
- Empty or missing data: check you are using the expected DB path (`--db /path/to/kcal.db`).
- Lookup/provider failures: verify env vars (`KCAL_USDA_API_KEY`, `KCAL_UPCITEMDB_API_KEY`) and provider selection.
- Import issues: rerun with `--dry-run` and inspect parse/column errors before applying.
- Recovery issues: run `kcal doctor`, then use [Backups and Recovery](#backups-and-recovery).

---

## Installation

Install `kcal` and confirm the binary runs.

### Most Common Commands

```bash
brew tap saadjs/kcal
brew install kcal
```

```bash
go install github.com/saad/kcal-cli@latest
kcal --help
```

### Advanced

Build from source:

```bash
git clone https://github.com/saad/kcal-cli.git
cd kcal-cli
go build -o kcal .
./kcal --help
```

---

## Getting Started

Run a minimal end-to-end flow and verify daily reporting works.

### Most Common Commands

```bash
kcal init
kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-20
kcal entry add --name "Chicken bowl" --calories 550 --protein 45 --carbs 40 --fat 18 --category lunch
kcal today
kcal analytics week
```

### Advanced

```bash
kcal category add snacks
kcal entry quick "Greek Yogurt | 220 20 18 8 | breakfast"
kcal goal current
kcal body add --weight 172 --unit lb --date 2026-02-20 --time 07:00
```

---

## Command Reference

Find command families quickly, then drill into `--help` for full flag details.

### In This Section

- [Top-Level Commands](#top-level-commands)
- [Nutrition Logging](#nutrition-logging)
- [Goals and Body](#goals-and-body)
- [Recipes and Exercise](#recipes-and-exercise)
- [Analytics](#analytics)
- [Data and Integrity](#data-and-integrity)
- [Lookup and Providers](#lookup-and-providers)

### Top-Level Commands

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

### Nutrition Logging

- `kcal category add|list|rename|delete`
- `kcal entry add|quick|list|show|update|metadata|delete|search|repeat`

```bash
kcal entry quick "Oats | 300 12 45 8 | breakfast" --date 2026-02-20 --time 08:00
kcal entry search --query oats --limit 10
kcal entry repeat 12 --date 2026-02-21 --time 08:00
```

### Goals and Body

- `kcal goal set|current|history|suggest`
- `kcal body add|list|update|delete`
- `kcal body-goal set|current|history`

```bash
kcal goal suggest --weight 80 --unit kg --maintenance-calories 2500 --pace cut --apply --effective-date 2026-02-20
kcal body-goal set --target-weight 170 --unit lb --target-body-fat 18 --effective-date 2026-02-20
```

### Recipes and Exercise

- `kcal recipe add|list|show|update|delete|log|recalc`
- `kcal recipe ingredient add|list|update|delete`
- `kcal exercise add|list|update|delete`

```bash
kcal recipe add --name "Overnight oats" --calories 0 --protein 0 --carbs 0 --fat 0 --servings 2
kcal recipe ingredient add "Overnight oats" --name Oats --amount 40 --unit g --calories 150 --protein 5 --carbs 27 --fat 3
kcal recipe recalc "Overnight oats"
kcal recipe log "Overnight oats" --servings 1 --category breakfast
```

### Analytics

- `kcal analytics week|month|range`
- `kcal analytics insights week|month|range`

```bash
kcal analytics month --month 2026-02
kcal analytics range --from 2026-02-01 --to 2026-02-20
kcal analytics insights range --from 2026-02-01 --to 2026-02-20 --granularity auto --out insights.md --out-format markdown
```

### Data and Integrity

- `kcal backup create|list|restore`
- `kcal export --format json|csv --out <file>`
- `kcal import --format json|csv --in <file> [--mode fail|skip|merge|replace] [--dry-run]`
- `kcal doctor [--fix]`

```bash
kcal export --format json --out backup.json
kcal import --format json --in backup.json --mode merge --dry-run
kcal doctor --fix
```

### Lookup and Providers

- `kcal lookup barcode <code>`
- `kcal lookup search --query <text> [--fallback --limit N --verified-only --verified-min-score X]`
- `kcal lookup providers`
- `kcal lookup usda-help|openfoodfacts-help|upcitemdb-help`
- `kcal lookup override set|show|list|delete`
- `kcal lookup cache list|purge|refresh`
- `kcal lookup cache search-list|search-purge`

```bash
kcal lookup barcode 3017620422003 --provider openfoodfacts
kcal lookup search --query "greek yogurt" --fallback --limit 10 --verified-only
kcal lookup override list --provider openfoodfacts
kcal lookup cache purge --provider openfoodfacts --all
kcal lookup cache search-list --provider openfoodfacts --query "greek yogurt"
```

For flag-level details:

```bash
kcal <command> --help
```

---

## Config and Paths

Control DB location and provider configuration precedence.

### Most Common Commands

```bash
kcal --db /tmp/kcal.db init
kcal --db /tmp/kcal.db today
```

```bash
kcal config set --barcode-provider openfoodfacts
kcal config set --fallback-order openfoodfacts,usda,upcitemdb
kcal config get
```

### Advanced

Environment variables:

- `KCAL_USDA_API_KEY`
- `KCAL_BARCODE_API_KEY` (legacy fallback)
- `KCAL_UPCITEMDB_API_KEY`
- `KCAL_UPCITEMDB_KEY_TYPE`
- `KCAL_BARCODE_PROVIDER`
- `KCAL_BARCODE_FALLBACK_ORDER`

Resolution notes:

- Provider can come from `--provider`, config, or env var defaults.
- API keys can come from command flags or env vars.
- Local override and cache layers can satisfy lookup before live provider calls.

See also:
- [Barcode Providers](#barcode-providers)
- [Lookup Overrides and Cache](#lookup-overrides-and-cache)

---

## Barcode Providers

Configure external nutrition lookup providers and fallback behavior.

### In This Section

- [Supported Providers](#supported-providers)
- [Basic Lookup Examples](#basic-lookup-examples)
- [USDA Setup](#usda-setup)
- [Open Food Facts Setup](#open-food-facts-setup)
- [UPCitemdb Setup](#upcitemdb-setup)
- [Fallback Order](#fallback-order)
- [Text Search and Verification](#text-search-and-verification)

### Supported Providers

- `usda` (default)
- `openfoodfacts`
- `upcitemdb`

```bash
kcal lookup providers
```

### Basic Lookup Examples

```bash
kcal lookup barcode 012345678905 --provider usda
kcal lookup barcode 3017620422003 --provider openfoodfacts
kcal lookup barcode 012993441012 --provider upcitemdb
```

### USDA Setup

```bash
export KCAL_USDA_API_KEY=your_key_here
kcal lookup usda-help
kcal lookup barcode 786012004549 --provider usda
```

### Open Food Facts Setup

```bash
export KCAL_BARCODE_PROVIDER=openfoodfacts
kcal lookup openfoodfacts-help
kcal lookup barcode 3017620422003
```

### UPCitemdb Setup

```bash
export KCAL_UPCITEMDB_API_KEY=your_key_here
export KCAL_UPCITEMDB_KEY_TYPE=3scale
kcal lookup upcitemdb-help
kcal lookup barcode 012993441012 --provider upcitemdb
```

### Fallback Order

```bash
export KCAL_BARCODE_FALLBACK_ORDER="usda,openfoodfacts,upcitemdb"
kcal lookup barcode 012345678905 --fallback
```

### Text Search and Verification

```bash
kcal lookup search --query "greek yogurt" --fallback --limit 10
kcal lookup search --query "greek yogurt" --fallback --verified-only --verified-min-score 0.85
```

- Search responses include `confidence_score` and `is_verified`.
- `--verified-min-score` controls the threshold used by `--verified-only`.

See also:
- [Lookup Overrides and Cache](#lookup-overrides-and-cache)
- [Config and Paths](#config-and-paths)

---

## Lookup Overrides and Cache

Use local overrides and cache controls for deterministic lookup results.

### Most Common Commands

```bash
kcal lookup override show 3017620422003 --provider openfoodfacts
kcal lookup override list --provider openfoodfacts
kcal lookup cache list --provider openfoodfacts --limit 50
kcal lookup cache refresh 3017620422003 --provider openfoodfacts
```

### Advanced

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

Delete override and purge cache:

```bash
kcal lookup override delete 3017620422003 --provider openfoodfacts
kcal lookup cache purge --provider openfoodfacts --all
kcal lookup cache search-purge --provider openfoodfacts --query "greek yogurt"
kcal lookup cache search-purge --all
```

Resolution order:
- Local override
- Local cache
- Live provider

See also:
- [Barcode Providers](#barcode-providers)
- [Command Reference](#command-reference)

---

## Import and Export

Move `kcal` data between environments using JSON or CSV.

### Most Common Commands

```bash
kcal export --format json --out kcal-export.json
kcal import --format json --in kcal-export.json --mode merge --dry-run
kcal import --format json --in kcal-export.json --mode merge
```

```bash
kcal export --format csv --out kcal-export.csv
kcal import --format csv --in kcal-export.csv
```

### Advanced

JSON import modes:

- `fail`: stop on conflicts.
- `skip`: ignore conflicting records.
- `merge`: merge compatible updates.
- `replace`: replace conflicting records.

Safety workflow:

```bash
kcal backup create
kcal import --format json --in kcal-export.json --mode merge --dry-run
kcal import --format json --in kcal-export.json --mode merge
kcal doctor
```

See also:
- [Backups and Recovery](#backups-and-recovery)
- [Development](#development)

---

## Backups and Recovery

Create recoverable snapshots and restore safely after incidents or migration mistakes.

### In This Section

- [Create Backups](#create-backups)
- [List Backups](#list-backups)
- [Restore Backups](#restore-backups)
- [Integrity Checks](#integrity-checks)
- [Recovery Runbook](#recovery-runbook)

### Create Backups

```bash
kcal backup create
kcal backup create --out /tmp/kcal-2026-02-20.db
kcal backup create --dir /tmp/kcal-backups
```

### List Backups

```bash
kcal backup list
kcal backup list --dir /tmp/kcal-backups
```

### Restore Backups

```bash
kcal backup restore --file /tmp/kcal-2026-02-20.db
kcal backup restore --file /tmp/kcal-2026-02-20.db --force
```

### Integrity Checks

```bash
kcal doctor
kcal doctor --fix
```

### Recovery Runbook

```bash
kcal backup list
kcal backup restore --file /path/to/known-good.db --force
kcal doctor
```

Recovery notes:

- Restore fails when `--file` is omitted or path is invalid.
- Restore may refuse overwrite without `--force`.
- `doctor --fix` applies only safe autofixes; unresolved issues can remain.

See also:
- [Import and Export](#import-and-export)
- [Releases](#releases)

---

## Analytics and Goals

Track adherence and trends across weekly, monthly, and custom date ranges.

### Most Common Commands

```bash
kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-01
kcal goal current
kcal goal history
kcal analytics week
kcal analytics month --month 2026-02
```

```bash
kcal analytics range --from 2026-02-01 --to 2026-02-20
kcal analytics insights week
kcal analytics insights range --from 2026-02-01 --to 2026-02-20 --granularity auto --out insights.md --out-format markdown
```

### Advanced

```bash
kcal goal suggest --weight 80 --unit kg --maintenance-calories 2500 --pace cut --apply --effective-date 2026-02-20
kcal body add --weight 172 --unit lb --body-fat 20 --date 2026-02-20 --time 07:00
kcal body-goal set --target-weight 170 --unit lb --target-body-fat 18 --effective-date 2026-02-20
```

Interpretation notes:

- Standard analytics reports summarize intake, exercise, net calories, category breakdowns, and adherence.
- Insights include period-over-period deltas, consistency metrics, streaks, and optional chart output.
- Exercise-adjusted adherence compares against effective targets that include logged exercise.

See also:
- [Getting Started](#getting-started)
- [Command Reference](#command-reference)

---

## Development

Local contributor workflow for testing, formatting, and docs maintenance.

### Most Common Commands

```bash
git clone https://github.com/saad/kcal-cli.git
cd kcal-cli
go mod download
go test ./...
```

### Advanced

```bash
gofmt -l $(find . -name '*.go' -not -path './vendor/*')
go vet ./...
```

This repo currently uses GitHub Actions for:
- Release publishing on version tags.
- Homebrew formula sync after release.
- GitHub Pages deployment for docs.

Docs maintenance:
- Keep README concise and route advanced content to this page.
- Validate internal docs links after edits.
- Keep examples aligned with command definitions in `cmd/kcal/*.go`.

See also:
- [Releases](#releases)
- [Command Reference](#command-reference)

---

## Releases

Publish tagged multi-platform binaries and keep release assets/checksums traceable.

### Most Common Commands

```bash
go test ./...
git tag v0.1.0
git push origin v0.1.0
```

### Advanced

Release workflow builds:
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

Artifacts are published with generated `checksums.txt`.

Before tagging:
- Confirm README command map matches current top-level commands.
- Confirm docs examples still match `cmd/kcal/*.go` behavior.
- Track notable changes in [`CHANGELOG.md`](https://github.com/saad/kcal-cli/blob/main/CHANGELOG.md).
