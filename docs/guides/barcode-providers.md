---
title: Barcode Providers
---

# Barcode Providers

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

- [Lookup Overrides and Cache](/kcal/guides/lookup-overrides-and-cache/)
- [Config and Paths](/kcal/reference/config-and-paths/)

## Failure and Edge Cases

- USDA lookups fail without a key unless another provider is used.
- Provider APIs may throttle requests; use cache and fallback to improve reliability.
- Invalid barcode formats can return no-match responses even with valid provider configuration.
