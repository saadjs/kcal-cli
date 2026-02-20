---
title: Development
---

# Development

Purpose: Document local developer workflow, checks, and CI expectations.

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

## CI Summary

Current CI workflow validates:
- Go setup via `go.mod` version
- `gofmt` formatting check
- `go vet ./...`
- `go test ./...`

## Docs Maintenance

- Keep README concise and route advanced content to `/docs` pages.
- Validate internal docs links after edits.
- Keep examples aligned with command definitions in `cmd/kcal/*.go`.

## See Also

- [Releases](/kcal/releases/)
- [Command Reference](/kcal/reference/commands/)

## Failure and Edge Cases

- If `go test ./...` fails at compile time, resolve branch-level API mismatches first.
- `gofmt` output listing files indicates formatting drift that CI will reject.
- Doc links using absolute `/kcal/...` paths must match `baseurl` in `_config.yml`.
