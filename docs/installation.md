---
title: Installation
---

# Installation

Purpose: Install `kcal` locally for day-to-day CLI usage.

When to use this page:
- You are installing `kcal` for the first time.
- You are choosing between source install and Homebrew.

## Option 1: Homebrew

```bash
brew tap saadjs/kcal
brew install kcal
```

## Option 2: Go Install

```bash
go install github.com/saad/kcal-cli@latest
kcal --help
```

## Option 3: Build from Source

```bash
git clone https://github.com/saad/kcal-cli.git
cd kcal-cli
go build -o kcal .
./kcal --help
```

## Next Step

- Continue to [Getting Started](/kcal-cli/getting-started/).

## Failure and Edge Cases

- If `go install` succeeds but `kcal` is not found, add your Go bin directory to `PATH`.
- If local build fails, run `go mod download` and retry.
- If Homebrew install fails, verify tap access and run `brew update` before retrying.
