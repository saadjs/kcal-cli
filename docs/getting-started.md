---
title: Getting Started
---

# Getting Started

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

- [Command Reference](/kcal-cli/reference/commands/)
- [Analytics and Goals](/kcal-cli/guides/analytics-and-goals/)
- [Barcode Providers](/kcal-cli/guides/barcode-providers/)

## Failure and Edge Cases

- `kcal goal set` requires full macro inputs; verify all required flags are provided.
- Invalid date formats must be `YYYY-MM-DD`.
- If you use custom DB locations, append `--db /path/to/kcal.db`.
