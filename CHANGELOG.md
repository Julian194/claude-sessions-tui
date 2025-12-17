# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


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
