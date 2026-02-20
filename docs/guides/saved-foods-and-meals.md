---
title: Saved Foods and Meals
---

# Saved Foods and Meals

Purpose: Build and reuse first-class food and meal templates for faster logging.

## Saved Food quick flow

```bash
kcal saved-food add --name "Greek Yogurt" --calories 150 --protein 15 --carbs 10 --fat 5 --category breakfast
kcal saved-food log "Greek Yogurt" --servings 2 --date 2026-02-20 --time 08:00
```

## Create from barcode

```bash
kcal saved-food add-from-barcode 3017620422003 --provider openfoodfacts --category snacks
```

## Create from existing entry

```bash
kcal entry list --limit 5
kcal saved-food add-from-entry 12
```

## Saved Meal quick flow

```bash
kcal saved-meal add --name "Yogurt bowl" --category breakfast
kcal saved-meal component add "Yogurt bowl" --saved-food "Greek Yogurt"
kcal saved-meal component add "Yogurt bowl" --name "Granola" --quantity 30 --unit g --calories 140 --protein 3 --carbs 22 --fat 5
kcal saved-meal log "Yogurt bowl" --servings 1.5
```

## Archive and restore

```bash
kcal saved-food archive "Greek Yogurt"
kcal saved-food list
kcal saved-food list --include-archived
kcal saved-food restore "Greek Yogurt"
```
