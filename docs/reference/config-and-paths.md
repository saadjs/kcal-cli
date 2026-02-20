---
title: Config and Paths
---

# Config and Paths

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
- [Barcode Providers](/kcal-cli/guides/barcode-providers/)
- [Lookup Overrides and Cache](/kcal-cli/guides/lookup-overrides-and-cache/)

## Failure and Edge Cases

- Invalid fallback-order values cause lookup validation failures.
- Missing provider key material causes provider-specific errors.
- Mismatched DB paths can make config appear missing.
