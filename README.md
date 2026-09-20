# Stacked C4 SVG

Turn a set of PlantUML C4 diagrams into a single portable SVG that is its own viewer. Open the file in any browser and switch between the context, container, component and code levels, drill down by clicking elements, resize, hide notes, and highlight or pin the paths you are following. No server, no external assets.

<p align="center">
  <a href="./docs/demo.png"><img src="./docs/demo-thumb.png?raw=true" width="400" alt="Example screenshot"></a>
</p>

## Features

The output is a viewer, not just a diagram. Everything below is embedded in the one SVG file.

- **Self-contained**: one file holding all diagrams and the viewer script, with no external assets
- **Level switching**: navigation buttons for each C4 level present, with 3 or 4 levels supported (code level optional)
- **Drill-down**: click a diagram element with a `$link` to move to the next level
- **Sizing modes**: native size with browser scrollbars, or auto-scale to fit the viewport
- **Note toggling**: hide or show PlantUML notes and the paths attached to them
- **Path highlighting**: hover a path label to bring that path to the front and highlight it
- **Pinned selection**: click a path label to keep it highlighted, Ctrl-click (Cmd-click on macOS) to pin several, Escape to clear

## Quick Start

1. **Create your PlantUML C4 diagrams** with numbered prefixes:
   ```
   01-context.puml
   02-container.puml
   03-component.puml
   04-code.puml (optional)
   ```

2. **Generate the stacked SVG**:
   ```bash
   ./manage.sh generate <directory> > output.svg

   # With custom title
   ./manage.sh generate <directory> --title "My System" > output.svg

   # With output file
   ./svg-stacker <directory> --output output.svg --title "My System"
   ```

3. **View the result**: Open the generated SVG in your browser

## Generating the Diagrams with Claude Code

[`docs/c4-diagram-prompt.md`](docs/c4-diagram-prompt.md) holds the instructions for having Claude Code write the `.puml` files for a project. Paste them into Claude Code with the placeholders filled in.

## How It Works

1. When the directory holds `.puml` files, renders them with PlantUML into a temporary directory
2. Parses each numbered SVG once, reading its dimensions and rewriting the content: `<script>` and `<title>` elements go, `$link` anchors become drill-down click handlers, and notes and the paths attached to them are tagged with classes
3. Renders one document from a template: header, level buttons, one hidden layer per level, and the embedded viewer script
4. Writes the single self-contained file to standard output or `--output`

## Requirements

- Go compiler for building the generator
- PlantUML on `PATH` when the input directory holds `.puml` files. Its bundled C4 library is used.
- Chromium on `PATH` only to run the viewer tests; they skip without it

## Usage

### Using manage.sh (recommended)

```bash
# Build
./manage.sh build

# Test
./manage.sh test

# Generate with default title
./manage.sh generate examples/ > example.svg

# Generate with custom title
./manage.sh generate examples/ --title "My Architecture" > example.svg

# Generate to file with title
./manage.sh generate examples/ --output output.svg --title "My System"
```

### Using svg-stacker directly

```bash
# Output to stdout
./svg-stacker <directory> > output.svg

# Output to file
./svg-stacker <directory> --output output.svg

# With custom title
./svg-stacker <directory> --output output.svg --title "My System"
```

## File Naming

Files must be numbered 01-04 with the following convention:
- `01-*.puml` - Context diagram (required)
- `02-*.puml` - Container diagram (required)
- `03-*.puml` - Component diagram (required)
- `04-*.puml` - Code diagram (optional)

Example: `01-context.puml`, `02-container.puml`, `03-component.puml`, `04-code.puml`

A directory of already-rendered `.svg` files follows the same convention: `01-*.svg` to `04-*.svg`. Files without a numeric prefix are ignored.

## Adding Clickable Navigation

To enable drill-down navigation, add `$link` parameter to your PlantUML elements:

```plantuml
System(my_system, "My System", "Description", $link="02-container.svg")
Container(my_container, "Container", "Tech", "Description", $link="03-component.svg")
Component(my_component, "Component", "Description", $link="04-code.svg")
```

The link filenames don't matter - they're replaced with JavaScript navigation.

## Project Structure

- `main.go` - the converter
- `stacked.svg.tmpl` - the output document template, embedded in the binary
- `navigation.js` - the viewer script, embedded in the binary
- `*_test.go` - unit, golden-file and headless-browser tests
- `testdata/` - golden cases: numbered input SVGs and the `expected.svg` for each
- `examples/` - example PlantUML sources and the generated `example.svg`
- `docs/c4-diagram-prompt.md` - instructions for generating the diagrams with Claude Code
- `C4-DIAGRAM-SPEC.md` - the diagram conventions those instructions refer to
- `manage.sh` - build, test, generate, release and clean script
- `svg-stacker` - compiled Go binary (gitignored)
- `CLAUDE.md` - development guidance for Claude Code

