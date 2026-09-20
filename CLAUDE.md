# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Purpose

`svg-stacker` folds a set of PlantUML C4 diagrams into one self-contained SVG that is its own viewer: level switching, drill-down, sizing modes, note toggling, and path highlighting with pinned selection, all in a browser with no server. The Go binary is the converter; the embedded script is the viewer.

## Commands

- **Build**: `go build -o svg-stacker` (or `./manage.sh build`)
- **Test**: `go test ./...` (or `./manage.sh test` for coverage). Viewer tests need Chromium on `PATH` and skip otherwise.
- **Update golden expectations** after an intentional output change: `go test -update ./...`, then review the diff under `testdata/`.
- **Generate**: `./svg-stacker <directory> [--output FILE] [--title TITLE]`. `plantuml` must be on `PATH` when the directory holds `.puml` files.
- **Regenerate the example**: `./svg-stacker examples/ --output examples/example.svg`

## Layout

- `main.go`: the whole converter, ordered so declarations with fewer in-file dependencies come first and `main` is last
- `stacked.svg.tmpl`: the output document as a `text/template`; everything static about the output lives here
- `navigation.js`: the viewer script, embedded into the document
- `main_test.go`, `parse_test.go`: unit tests for discovery, argument parsing and the SVG rewrite
- `golden_test.go`: generates each `testdata/<case>/` and compares with its `expected.svg`
- `viewer_test.go`: runs the generated document in headless Chromium and asserts on DOM state
- `examples/`: PlantUML sources and `example.svg`, the generated result
- `docs/c4-diagram-prompt.md`: the prompt for having Claude Code write the `.puml` files
- `C4-DIAGRAM-SPEC.md`: the diagram conventions that prompt refers to
- `manage.sh`: build, test, generate, release and clean targets
- `svg-stacker`: the built binary, gitignored

## Pipeline

`run` composes three pure steps: `render` turns `.puml` files into SVGs in a temporary directory, `load` discovers and parses one `Diagram` per level, and `stack` renders the template for a title, a timestamp and the diagrams. `parse_diagram` decodes an input once with `encoding/xml` into a token slice, reads the dimensions from the root element, rewrites the content and re-encodes it. No regexes touch XML.

## Level convention

The C4 levels are defined once, in the `levels` table in `main.go`, and every consumer derives order from it. Input files are matched by numeric prefix on the base name, for `.puml` and `.svg` alike:

| Prefix | Level | Required |
|---|---|---|
| `01-` | context | yes |
| `02-` | container | yes |
| `03-` | component | yes |
| `04-` | code | no |

Drill-down comes from PlantUML's `$link` parameter on an element; the target filename is irrelevant because the converter replaces the link with viewer navigation.

## Converter-to-viewer seam

The viewer script relies on exactly these IDs and classes, which the converter emits:

| Emitted by converter | Read by viewer |
|---|---|
| `id="layer-<level>"` on each layer group | show and hide |
| `id="container-<level>"` on the layer's backing rect | resize |
| `id="diagram-<level>"` on the group wrapping the nested `svg` | resize |
| `id="nav-<level>"` on each button rect | active-button styling |
| `id="fit-toggle"`, `id="fit-text"`, `id="notes-toggle"`, `id="notes-text"` | button positioning and text |
| `class="note"` on note groups | note toggling |
| `class="note-link"` on links attached to notes | note toggling |
| `class="link"` on link groups (as emitted by PlantUML) | hover and pin |
| `onclick="navigateDown()"` on linked groups | drill-down |
| `const diagramData = {...}` and `const availableLevels = [...]` | resize and navigation |

Static rewriting belongs in the converter. The script does only what needs layout or interaction, and enhances each layer once, on first show.

## Dependencies

Go standard library only. `plantuml` is needed at run time for `.puml` input; Chromium is needed only to run the viewer tests.
