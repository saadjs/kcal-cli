# kcal-cli

`kcal` is a local-first calorie, macro, and micronutrient tracking CLI built with Go, Cobra, and SQLite.

## Features

- Meal entry CRUD for calories, macros, and richer nutrients (fiber, sugar, sodium, micronutrients)
- Default + custom categories (`breakfast`, `lunch`, `dinner`, `snacks`, and custom like `supper`)
- Goal versioning by effective date
- Body tracking (weight + optional body-fat) with body-goal versioning
- Recipe CRUD with serving-based logging
- Ingredient-level recipe builder with recipe total recalculation
- Exercise log CRUD with burned calories and activity details
- Weekly/monthly/custom-range analytics with intake/exercise/net calories and exercise-adjusted adherence targets
- Premium-style insights (`analytics insights`) with period-over-period deltas, consistency stats, and terminal charts

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
./kcal entry add --name "Chicken bowl" --calories 550 --protein 45 --carbs 40 --fat 18 --fiber 6 --sugar 4 --sodium 580 --micros-json '{"vitamin_c":{"value":40,"unit":"mg"}}' --category supper
./kcal entry add --barcode 3017620422003 --provider openfoodfacts --servings 1.5 --category snacks
./kcal recipe add --name "Overnight oats" --calories 0 --protein 0 --carbs 0 --fat 0 --servings 2
./kcal recipe ingredient add "Overnight oats" --name Oats --amount 40 --unit g --calories 150 --protein 5 --carbs 27 --fat 3
./kcal recipe ingredient add "Overnight oats" --name Milk --amount 200 --unit ml --calories 80 --protein 5 --carbs 10 --fat 2
# Scaling helper example (auto-scale from reference macros; density needed for volume<->mass)
./kcal recipe ingredient add "Overnight oats" --name PeanutButter --amount 2 --unit tbsp --ref-amount 32 --ref-unit g --ref-calories 190 --ref-protein 7 --ref-carbs 8 --ref-fat 16 --density-g-per-ml 1.05
./kcal recipe recalc "Overnight oats"
./kcal recipe log "Overnight oats" --servings 1 --category breakfast
./kcal exercise add --type running --calories 300 --duration-min 35 --date 2026-02-20 --time 18:30
./kcal analytics week
./kcal analytics insights range --from 2026-02-01 --to 2026-02-20 --granularity auto
```

## Command Reference

- `kcal init`
- `kcal category add|list|rename|delete`
- `kcal entry add|list|update|delete`
  - `kcal entry show <id>`
  - `kcal entry metadata <id> --metadata-json '{...}'`
  - `entry add` supports `--metadata-json '{...}'`, `--fiber`, `--sugar`, `--sodium`, `--micros-json '{...}'`
  - `entry list` supports `--with-metadata` and `--with-nutrients`
- `kcal goal set|current|history`
- `kcal body add|list|update|delete`
- `kcal body-goal set|current|history`
- `kcal exercise add|list|update|delete`
- `kcal recipe add|list|show|update|delete|log|recalc`
- `kcal recipe ingredient add|list|update|delete`
- `kcal analytics week|month|range`
- `kcal analytics insights week|month|range [--granularity auto|day|week|month] [--no-charts] [--json] [--out <file>] [--out-format text|markdown|json]`
- `kcal lookup barcode <code> [--provider usda|openfoodfacts|upcitemdb] [--api-key ...] [--json]`
- `kcal lookup providers`
- `kcal lookup usda-help`
- `kcal lookup openfoodfacts-help`
- `kcal lookup upcitemdb-help`
- `kcal lookup override set|show|list|delete ...`
- `kcal lookup cache list|purge|refresh ...`
- `kcal config set|get`
- `kcal export --format json|csv --out <file>`
- `kcal import --format json|csv --in <file>`

Use `--help` on any command for details.

### Exercise-Adjusted Goal Logic

- Exercise increases calorie allowance for the day (`effective_goal_calories = goal_calories + exercise_calories`).
- Exercise also increases macro targets proportionally using the goal's calorie split (protein/carbs/fat by 4/4/9 kcal per gram).
- Adherence uses net calories (`intake - exercise`) and exercise-adjusted macro targets.

## Barcode Lookup Providers

```bash
export KCAL_USDA_API_KEY=your_key_here
./kcal lookup barcode 012345678905 --provider usda
./kcal lookup barcode 3017620422003 --provider openfoodfacts
./kcal lookup barcode 012993441012 --provider upcitemdb
./kcal lookup providers
```

API key resolution order:
- `--api-key` flag
- `KCAL_USDA_API_KEY`
- `KCAL_BARCODE_API_KEY` (legacy fallback)
- `KCAL_UPCITEMDB_API_KEY` (optional for UPCitemdb paid plans)
- Provider default can be set with `KCAL_BARCODE_PROVIDER` (`usda`, `openfoodfacts`, or `upcitemdb`)

Rate limit note:
- USDA default limit is `1,000 requests/hour/IP`.
- Open Food Facts uses fair-use limits and requires a descriptive user-agent.
- UPCitemdb published plan examples: Trial `100/day`, DEV `20,000 lookup/day + 2,000 search/day`, PRO `150,000 lookup/day + 20,000 search/day`.
- `kcal` caches barcode lookups in SQLite to reduce repeated provider calls.

Local correction workflow:
```bash
./kcal lookup override set 3017620422003 --provider openfoodfacts --name "Nutella Custom" --brand Ferrero --serving-amount 15 --serving-unit g --calories 99 --protein 1 --carbs 10 --fat 6 --fiber 0.5 --sugar 10 --sodium 15 --micros-json '{"vitamin_e":{"value":1.2,"unit":"mg"}}'
./kcal lookup barcode 3017620422003 --provider openfoodfacts
```
Override resolution order is: local override -> cache -> live provider.

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
