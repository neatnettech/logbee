# Contributing to Logbee

Thanks for your interest in improving Logbee.

## Development

You need Go 1.18 or later.

```bash
make build   # build the ./logbee binary
make test    # go test ./...
make vet     # go vet ./...
make lint    # golangci-lint (if installed)
```

Format code with `gofmt` (`make fmt`) before committing. The codebase uses tabs
and standard-library dependencies only — avoid adding heavy dependencies.

## Branching & commits

- `main` is the trunk; all work lands there via pull request.
- Keep pull requests focused and include tests where it makes sense.
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org)
  (`feat:`, `fix:`, `docs:`, `chore:`, …).

## Releases & versioning

This project follows [Semantic Versioning](https://semver.org).

- Release candidates are tagged `vX.Y.Z-rc.A` (published as GitHub pre-releases).
- Final releases are tagged `vX.Y.Z`.

Pushing a `v*` tag triggers the release workflow (GoReleaser), which builds the
cross-platform binaries and publishes the GitHub release. Update
[`CHANGELOG.md`](CHANGELOG.md) in the same PR that prepares a release.

## Reporting issues

Use the issue templates for bug reports and feature requests.
