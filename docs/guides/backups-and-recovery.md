---
title: Backups and Recovery
---

# Backups and Recovery

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

- [Import and Export](/kcal-cli/guides/import-export/)
- [Releases](/kcal-cli/releases/)

## Failure and Edge Cases

- Restore fails when `--file` is omitted or path is invalid.
- Restore may refuse overwrite without `--force`.
- `doctor --fix` applies only safe autofixes; unresolved issues can remain.
