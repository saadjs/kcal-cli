---
title: Command Reference
---

# Command Reference

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
