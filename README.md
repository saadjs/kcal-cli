# kcal-cli

`kcal` is a local-first calorie and macro tracking CLI built with Go, Cobra, and SQLite.

## Features

- Meal entry CRUD for calories and macros (protein, carbs, fat)
- Default + custom categories (`breakfast`, `lunch`, `dinner`, `snacks`, and custom like `supper`)
- Goal versioning by effective date
- Recipe CRUD with serving-based logging
- Weekly/monthly/custom-range analytics with adherence and category breakdown

## Install

```bash
go build -o kcal .
```

## Quickstart

```bash
./kcal init
./kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-01
./kcal category add supper
./kcal entry add --name "Chicken bowl" --calories 550 --protein 45 --carbs 40 --fat 18 --category supper
./kcal recipe add --name "Overnight oats" --calories 400 --protein 20 --carbs 50 --fat 10 --servings 2
./kcal recipe log "Overnight oats" --servings 1 --category breakfast
./kcal analytics week
```

## Command Reference

- `kcal init`
- `kcal category add|list|rename|delete`
- `kcal entry add|list|update|delete`
- `kcal goal set|current|history`
- `kcal recipe add|list|show|update|delete|log`
- `kcal analytics week|month|range`

Use `--help` on any command for details.

## Testing

```bash
go test ./...
```

## License

MIT (see `LICENSE`).
