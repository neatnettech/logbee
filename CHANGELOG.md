# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Pre-releases are tagged `vX.Y.Z-rc.A`; final releases `vX.Y.Z`.

## [Unreleased]

### Added
- `--interactive, -i <name>` (env `LOGBEE_INTERACTIVE`): forward your terminal's
  stdin to the named process so interactive dev servers (e.g. Expo/Metro) receive
  keypresses (`r`, `i`, `a`, `j`). The terminal is put in raw mode while running and
  restored on exit.
- Homebrew install via the `neatnettech/tap` tap (`brew install neatnettech/tap/logbee`).
  GoReleaser publishes the formula automatically on final `vX.Y.Z` releases.

## [1.2.0-rc.1] - 2026-05-30

First release of Logbee, a fork of [Hivemind](https://github.com/DarthSim/hivemind)
by neatnettech.

### Added
- `--log-file, -L` (env `LOGBEE_LOG_FILE`): write the aggregated output stream to
  a file, live and line-by-line, for `tail -f` or external analytics/LLM pipelines.
  The file is plain, greppable text — the `name | ` prefix carries no color escapes
  and ANSI codes are stripped from process output. Parent directories are created
  automatically. The console output is unchanged (colors intact).
- `--log-append` (env `LOGBEE_LOG_APPEND`): append to the log file instead of
  truncating it on start.

### Changed
- Rebranded from Hivemind to Logbee: module path `github.com/neatnettech/logbee`,
  binary `logbee`, and all environment variables `HIVEMIND_*` → `LOGBEE_*`.

[Unreleased]: https://github.com/neatnettech/logbee/compare/v1.2.0-rc.1...HEAD
[1.2.0-rc.1]: https://github.com/neatnettech/logbee/releases/tag/v1.2.0-rc.1
