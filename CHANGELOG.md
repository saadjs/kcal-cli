# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

### Added
- Initial `kcal` CLI implementation with local SQLite storage.
- Commands for init, categories, entries, goals, recipes, and analytics.
- Service-level and CLI-level regression test suites.
- CI workflow for formatting, vetting, and tests.
- Release workflow for multi-platform binary builds and GitHub Releases.
- Body measurement tracking (`kcal body`) with kg/lb support and optional body-fat percentage.
- Body-goal tracking (`kcal body-goal`) with effective-date versioning.
- Ingredient-level recipe builder (`kcal recipe ingredient ...`) and recipe total recalculation (`kcal recipe recalc`).
- Extended analytics body section with weight/body-fat/lean-mass trends and body-goal progress.
- Multi-provider barcode lookup (`usda`, `openfoodfacts`, `upcitemdb`) with local SQLite cache, provider selection, setup helper commands, published UPCitemdb plan-limit guidance, and local barcode nutrition overrides.
- Saved template database schema and model support for saved foods, saved meals, and saved meal components.
- Saved template command families: `kcal saved-food ...` and `kcal saved-meal ...` (including create from entry/barcode, component management, archive/restore, and logging).
- Saved templates included in JSON portability workflows (`kcal export --format json`, `kcal import --format json`), including coverage in portability tests.

### Changed
- Consolidated GitHub Pages docs into a single-page experience at `docs/index.md` and removed legacy multi-page duplicates from publish paths.
- Updated docs and README command maps/quick flows to include saved template workflows.
