---
title: Development
---

# Development

Purpose: Document local developer workflow and checks.

When to use this page:
- You are contributing code or docs.
- You are troubleshooting CI parity locally.

## Local Setup

```bash
git clone https://github.com/saad/kcal-cli.git
cd kcal-cli
go mod download
```

## Run Tests

```bash
go test ./...
```

## Formatting and Vet

```bash
gofmt -l $(find . -name '*.go' -not -path './vendor/*')
go vet ./...
```

## Automation Summary

This repo currently uses GitHub Actions for:
- Release publishing on version tags.
- Homebrew formula sync after release.
- GitHub Pages deployment for docs.

## Docs Maintenance

- Keep README concise and route advanced content to `/docs` pages.
- Validate internal docs links after edits.
- Keep examples aligned with command definitions in `cmd/kcal-cli/*.go`.

## See Also

- [Releases](/kcal-cli/releases/)
- [Command Reference](/kcal-cli/reference/commands/)

## Failure and Edge Cases

- If `go test ./...` fails at compile time, resolve branch-level API mismatches first.
- `gofmt` output listing files indicates formatting drift that should be fixed before publishing.
- Doc links using absolute `/kcal-cli/...` paths must match `baseurl` in `_config.yml`.
