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
- Multi-provider barcode lookup (`usda`, `openfoodfacts`) with local SQLite cache, provider selection, and setup helper commands.
