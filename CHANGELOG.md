# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2025-12-19

### Added
- Automatic stacked SVG generation after Claude diagram creation
- Multi-select path highlighting with Ctrl+click (or Cmd+click on Mac)
- Persistent path selection that remains highlighted until clicked again or ESC is pressed
- Invisible SVG metadata embedding with generator name, version, and timestamp

### Changed
- Project name detection now uses current working directory instead of git remote
- Improved release command to inject version via `-ldflags` at compile time

## [0.5.0] - 2025-10-21

### Added
- `prompt` subcommand for Claude Code integration: auto-analyzes project context and generates C4 diagrams with Claude
- PlantUML syntax validation instructions in C4 specification
- Embedded C4 specification in binary for self-contained prompt generation
- Release command now prompts for semver version during binary compilation

### Fixed
- Compile-time version value setting in release builds

## [0.4.0] - 2025-10-15

### Added
- Parallel PlantUML processing with `-nbthread auto` for faster diagram generation
- Pretty-printed SVG content embedding for improved readability

### Fixed
- SVG namespace handling bug that was mangling xlink attributes
- Note visibility now properly applies to associated path links

## [0.3.0] - 2025-10-14

### Added
- Path hover highlighting for better interactivity
- Clean task to manage.sh for removing generated artifacts

### Fixed
- Horizontal scrolling issues in diagram layers

## [0.2.0] - 2025-10-14

### Changed
- Embedded navigation.js directly in binary instead of external file reference

### Fixed
- Broken person icon rendering in C4 diagrams

## [0.1.0] - 2025-10-03

### Added
- Initial release with core functionality
- manage.sh build and release automation
- Multi-platform binary support (Linux/macOS/Windows, amd64/arm64)
