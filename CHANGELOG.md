# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [0.2.6] - 2025-12-19

### Added

- opencode: implement GetSlashCommands and BranchSession
- add model info to preview and exports
- add subagent/branch toggle and visual indicators
- add OpenCode sessions adapter
- implement TUI/CLI entry point (Phase 2.6-2.8)
- add Go core packages with tests (Phase 1)

### Changed

- improve TUI borders and add dev workflow docs
- implement incremental cache rebuild

### Fixed

- ci: lower coverage threshold to 60%
- ci: resolve test failures and coverage errors
- branch display showing under wrong parent in fzf
- add --no-sort to fzf to preserve branch grouping
- show formatted data immediately, skip rebuild if cache fresh
- restore branch display in TUI

## [0.2.5] - 2025-12-18

### Added

- add session branching feature

### Changed

- replace bash extraction with Python for 5x faster loading

### Fixed

- ci:  update test-pr path to scripts/
- ci:  remove references to moved scripts
- ignore date headers on enter key
- show last directory segment for root workspace sessions
- display orphaned branches correctly
- date separators now appear above their sessions
- centralize temp file cleanup in EXIT trap

## [Unreleased]

### Added

- Session branching feature (Ctrl-B) to create copies of sessions and explore alternative paths
- Branches are visually grouped under their parent sessions in the session list

### Fixed

- Centralize temp file cleanup in EXIT trap to prevent leftover files on script abort
- Date separators now appear above their sessions (not below)

## [0.2.4] - 2025-12-18

### Added

- show full tool details in markdown export
- click search result to scroll to message in full view
- show full tool details in HTML export
- add CI workflow with smoke tests

### Fixed

- correct homebrew install command in README, add upgrade script

## [0.2.3] - 2025-12-17

### Added

- add test-pr script for quick PR testing
- display slash command content in session exports
- add LLM-optimized markdown copy to clipboard (ctrl-y) (#1)

### Fixed

- auto-reset header after 1 second on copy/export actions
- auto-categorize changelog by commit prefix (feat→Added, fix→Fixed)

## [0.2.2] - 2025-12-12

### Fixed
- Improve release script with auto-changelog from commits
- Fix 0.2.1 changelog

## [0.2.1] - 2025-12-12

### Added
- `--dangerously-skip-permissions` flag on resume by default
- Release script for automated versioning

### Fixed
- Timestamp layout in HTML export (removed extra line)

## [0.2.0] - 2024-12-12

### Added
- Timestamps displayed on each message in HTML export (HH:MM format)

## [0.1.0] - 2024-12-12

### Added
- Initial release
- TUI browser with fzf for fuzzy search
- Session preview with topics, files, and stats
- HTML export with dark/light themes
- Session statistics (tokens, tools, cost estimates)
- Cross-platform support (macOS & Linux)
- Homebrew installation support
