# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Pre-releases are tagged `vX.Y.Z-rc.A`; final releases `vX.Y.Z`.

## [Unreleased]

### Added
- Homebrew install via the `neatnet/tap` tap (`brew install neatnet/tap/logbee`).
  GoReleaser publishes the formula automatically on final `vX.Y.Z` releases.

## [1.2.0-rc.1] - 2026-05-30

First release of Logbee, a fork of [Hivemind](https://github.com/DarthSim/hivemind)
by neatnet.

### Added
- `--log-file, -L` (env `LOGBEE_LOG_FILE`): write the aggregated output stream to
  a file, live and line-by-line, for `tail -f` or external analytics/LLM pipelines.
  The file is plain, greppable text — the `name | ` prefix carries no color escapes
  and ANSI codes are stripped from process output. Parent directories are created
  automatically. The console output is unchanged (colors intact).
- `--log-append` (env `LOGBEE_LOG_APPEND`): append to the log file instead of
  truncating it on start.

### Changed
- Rebranded from Hivemind to Logbee: module path `github.com/neatnet/logbee`,
  binary `logbee`, and all environment variables `HIVEMIND_*` → `LOGBEE_*`.

[Unreleased]: https://github.com/neatnet/logbee/compare/v1.2.0-rc.1...HEAD
[1.2.0-rc.1]: https://github.com/neatnet/logbee/releases/tag/v1.2.0-rc.1
