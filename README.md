# kcal-cli

`kcal` is a local-first calorie, macro, and nutrient tracking CLI built with Go and SQLite.

## Install

### Homebrew

```bash
brew tap saadjs/kcal
brew install kcal
```

### Go install

```bash
go install github.com/saad/kcal-cli@latest
```

### Build from source

```bash
go build -o kcal .
./kcal --help
```

## Quick Start

```bash
kcal init
kcal goal set --calories 2200 --protein 160 --carbs 240 --fat 70 --effective-date 2026-02-20
kcal entry add --name "Chicken bowl" --calories 550 --protein 45 --carbs 40 --fat 18 --category lunch
kcal today
kcal analytics week
```

## Command Map

Top-level commands:

- `analytics`
- `backup`
- `body`
- `body-goal`
- `category`
- `completion`
- `config`
- `doctor`
- `entry`
- `exercise`
- `export`
- `goal`
- `import`
- `init`
- `lookup`
- `recipe`
- `saved-food`
- `saved-meal`
- `today`

Use `kcal <command> --help` for command flags and subcommands.

## Saved Templates

Reusable templates for frequent foods and meals:

```bash
kcal saved-food add --name "Greek Yogurt" --calories 150 --protein 15 --carbs 10 --fat 5 --category breakfast
kcal saved-food add-from-barcode 3017620422003 --provider openfoodfacts --category snacks
kcal saved-meal add --name "Yogurt bowl" --category breakfast
kcal saved-meal component add "Yogurt bowl" --saved-food "Greek Yogurt"
kcal saved-meal log "Yogurt bowl" --servings 1
```

Saved foods and saved meals are also included in JSON import/export.

## Barcode Lookup (Essentials)

Set env vars as needed:

```bash
export KCAL_USDA_API_KEY=your_key_here
export KCAL_BARCODE_PROVIDER=openfoodfacts
export KCAL_BARCODE_FALLBACK_ORDER=usda,openfoodfacts,upcitemdb
# optional (paid UPCitemdb)
export KCAL_UPCITEMDB_API_KEY=your_key_here
```

Example lookup:

```bash
kcal lookup barcode 3017620422003 --provider openfoodfacts
kcal lookup search --query "greek yogurt" --fallback --limit 10 --verified-only
```

Search includes a weighted `confidence_score` and `is_verified` flag (default threshold `0.80`, configurable via `--verified-min-score`).
For compatibility, `provider_confidence` is still emitted and equals `confidence_score`.
Provider text-search cache can be managed with `kcal lookup cache search-list` and `kcal lookup cache search-purge`.

For provider setup, limits, overrides, and cache workflows, see docs below.

## Documentation

- GitHub Pages docs: <https://saadjs.github.io/kcal-cli/>
- Local docs index: [`docs/index.md`](docs/index.md)

## Development

```bash
go test ./...
```

Run checks locally before releases or major changes.

## Project Files

- Changelog: [`CHANGELOG.md`](CHANGELOG.md)
- License: [`LICENSE`](LICENSE)
