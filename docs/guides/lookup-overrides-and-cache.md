---
title: Lookup Overrides and Cache
---

# Lookup Overrides and Cache

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

- [Barcode Providers](/kcal/guides/barcode-providers/)
- [Command Reference](/kcal/reference/commands/)

## Failure and Edge Cases

- Override commands fail if required macro fields are omitted.
- Cache refresh fails if provider auth/config is incomplete.
- Purging with narrow filters may leave rows you still expect to clear.
