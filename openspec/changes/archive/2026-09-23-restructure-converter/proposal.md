## Why

The architecture review of 2026-09-20 found that `main.go` serves two purposes, models the four C4 levels three different ways, parses the same SVG with three different parsers, and leaves static document rewriting to the embedded viewer where it runs again on every level switch. The project is small enough that these are best resolved together, in a fixed order, with a regression check in place before any structure moves.

## What Changes

- Output becomes deterministic for a given input and timestamp, so a golden-file test can guard the rest of the work.
- **BREAKING** The `prompt` subcommand, `gatherProjectContext`, and the embedded `C4-DIAGRAM-SPEC.md` are removed from the binary. The spec file stays in the repository as documentation. The prompt the subcommand assembled is captured verbatim in `docs/c4-diagram-prompt.md`, with the project-context section written as placeholders, so a Claude Code skill can be built from it later.
- Dead code is removed: the unreachable "diagram not found" layer, `copyTestdataFiles`, and the unused fixtures in `testdata/`. The directory is reused for golden-file cases.
- The four C4 levels are defined once, in an ordered table pairing numeric prefix with level name. Discovery of `.puml` and `.svg` files, button order, layer order and the `availableLevels` array all derive from it. **BREAKING** `.svg` input files are matched by numeric prefix, the same as `.puml` files, not by substring.
- Each input SVG is parsed once with `encoding/xml`. Dimensions come from the `<svg>` start element. `<script>` and `<title>` elements are dropped, `<a>` wrappers become an `onclick` on the enclosing group, and note groups and note-attached links are tagged with a class.
- The converter becomes a pure pipeline: a `Diagram` value per level and a `stack(title, []Diagram) string` function rendering one `text/template`. `SVGStacker` and its mutable state are removed. PlantUML rendering is a separate function called once from `main`.
- The CLI uses the standard `flag` package. The interface is a directory argument, `--output` and `--title`.
- The viewer script does only layout-dependent work. Hitbox creation runs once per layer at load, the Escape listener is registered once, and `showLevel` only toggles visibility. Every viewer feature is kept.
- A first viewer test is added. It runs the generated document in headless Chromium from a Go test, skips when Chromium is absent, and establishes the pattern for testing the remaining viewer requirements.
- `CLAUDE.md` is rewritten to describe the current project.

## Capabilities

### New Capabilities
- `stacked-svg-converter`: the command line interface, level discovery, input SVG parsing, and the shape of the generated document, including the classes and IDs it promises to the viewer.
- `diagram-viewer`: the runtime behaviour of the embedded script: level switching, sizing modes, note toggling, path highlighting and pinned selection, and the guarantee that enhancement happens once per layer.

### Modified Capabilities
- none. `openspec/specs/` is empty.

## Impact

- `main.go` is rewritten. `main_test.go` is replaced by a golden-file test per example directory plus unit tests for parsing and level discovery.
- `navigation.js` is restructured: static rewriting is removed, enhancement is moved to load time.
- `viewer_test.go` and a small HTML harness under `testdata/` are added. Chromium is a test-time dependency only, and its absence skips the viewer tests.
- `docs/c4-diagram-prompt.md` is added.
- `C4-DIAGRAM-SPEC.md` is no longer embedded. `README.md` loses the "Generating C4 Diagrams with Claude" section. `CLAUDE.md` is rewritten.
- The `TODO.md` entry "Replace the Claude integration with a skill" is satisfied by the removal half of this change and by the captured prompt document; the skill itself is out of scope.
- No new dependencies. `go.mod` is unchanged.
