# kcal-cli

`kcal` is a local-first calorie and macro tracking CLI built with Go, Cobra, and SQLite.

## Features

- Meal entry CRUD for calories and macros (protein, carbs, fat)
- Default + custom categories (`breakfast`, `lunch`, `dinner`, `snacks`, and custom like `supper`)
- Goal versioning by effective date
- Body tracking (weight + optional body-fat) with body-goal versioning
- Recipe CRUD with serving-based logging
- Ingredient-level recipe builder with recipe total recalculation
- Weekly/monthly/custom-range analytics with adherence and category breakdown

## Install

```bash
go build -o kcal .
```

## Quickstart

```bash
./kcal init
./kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-01
./kcal body-goal set --target-weight 170 --unit lb --target-body-fat 18 --effective-date 2026-02-01
./kcal body add --weight 172 --unit lb --body-fat 20 --date 2026-02-20 --time 07:00
./kcal category add supper
./kcal entry add --name "Chicken bowl" --calories 550 --protein 45 --carbs 40 --fat 18 --category supper
./kcal recipe add --name "Overnight oats" --calories 0 --protein 0 --carbs 0 --fat 0 --servings 2
./kcal recipe ingredient add "Overnight oats" --name Oats --amount 40 --unit g --calories 150 --protein 5 --carbs 27 --fat 3
./kcal recipe ingredient add "Overnight oats" --name Milk --amount 200 --unit ml --calories 80 --protein 5 --carbs 10 --fat 2
# Scaling helper example (auto-scale from reference macros; density needed for volume<->mass)
./kcal recipe ingredient add "Overnight oats" --name PeanutButter --amount 2 --unit tbsp --ref-amount 32 --ref-unit g --ref-calories 190 --ref-protein 7 --ref-carbs 8 --ref-fat 16 --density-g-per-ml 1.05
./kcal recipe recalc "Overnight oats"
./kcal recipe log "Overnight oats" --servings 1 --category breakfast
./kcal analytics week
```

## Command Reference

- `kcal init`
- `kcal category add|list|rename|delete`
- `kcal entry add|list|update|delete`
- `kcal goal set|current|history`
- `kcal body add|list|update|delete`
- `kcal body-goal set|current|history`
- `kcal recipe add|list|show|update|delete|log|recalc`
- `kcal recipe ingredient add|list|update|delete`
- `kcal analytics week|month|range`

Use `--help` on any command for details.

## Testing

```bash
go test ./...
```

## CI and Releases

- CI runs `gofmt` verification, `go vet`, and `go test ./...` on pushes and pull requests.
- Tagging a version like `v0.1.0` triggers multi-platform binary builds and publishes a GitHub Release with artifacts and `checksums.txt`.
- Release process details are in `RELEASE_CHECKLIST.md`.

## License

MIT (see `LICENSE`).
