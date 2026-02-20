---
title: Import and Export
---

# Import and Export

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

- [Backups and Recovery](/kcal/guides/backups-and-recovery/)
- [Development](/kcal/development/)

## Failure and Edge Cases

- CSV imports fail when row column counts do not match supported formats.
- Invalid timestamps in CSV cause row-level parse errors.
- Imports can create category mismatches if category names are missing or malformed.
