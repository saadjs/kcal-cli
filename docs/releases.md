---
title: Releases
---

# Releases

Purpose: Publish tagged multi-platform binaries and keep release notes/checksums traceable.

When to use this page:
- You are preparing a versioned release.
- You need to understand automated release artifacts.

## Release Trigger

- Pushing tags matching `v*` triggers the release workflow.
- Manual dispatch is available for workflow testing.

## Build Matrix

Release workflow builds:
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

Artifacts are published with generated `checksums.txt`.

## Suggested Release Flow

```bash
go test ./...
git tag v0.1.0
git push origin v0.1.0
```

Then verify GitHub release assets and checksums.

## Checklist and History

- Track notable changes in [`CHANGELOG.md`](../CHANGELOG.md)

## Docs Drift Guard

Before tagging:
- Confirm README command map matches current top-level commands.
- Confirm docs examples still match `cmd/kcal-cli/*.go` behavior.

## Failure and Edge Cases

- Missing tag prefix `v` prevents automatic release publish stage.
- Partial artifact uploads can happen if one matrix leg fails.
- Checksum mismatches indicate corrupted or stale release outputs.
