# Stacked Svg Converter

## Purpose

Defines the command line contract, the input conventions, and the shape of the generated document that the converter produces from a directory of C4 diagrams.

## Requirements

### Requirement: Command line interface
The converter SHALL accept exactly one positional argument naming an input directory, an optional `--output` flag naming the file to write, and an optional `--title` flag setting the header text. With no `--output` the document SHALL be written to standard output. With no `--title` the header SHALL read "🏗️ Stacked C4 Architecture". The converter SHALL exit with a non-zero status and a usage message when the directory is missing or a flag is unknown. `-h`, `--help`, `-v` and `--version` SHALL be honoured.

#### Scenario: Output to standard output
- **WHEN** the converter is run with only a directory argument
- **THEN** the generated document is written to standard output and the exit status is zero

#### Scenario: Output to a file
- **WHEN** the converter is run with `--output out.svg`
- **THEN** the generated document is written to `out.svg` and nothing is written to standard output

#### Scenario: Unknown flag
- **WHEN** the converter is run with a flag it does not define
- **THEN** it prints a usage message to standard error and exits with a non-zero status

#### Scenario: Prompt subcommand removed
- **WHEN** the converter is run with the argument `prompt`
- **THEN** it treats `prompt` as a directory name and reports that no diagrams were found in it

### Requirement: Levels are discovered by numeric prefix
The converter SHALL recognise four levels in fixed order: `01` context, `02` container, `03` component, `04` code. An input file SHALL belong to a level when its base name starts with that level's two-digit prefix followed by a hyphen. Files without a recognised prefix SHALL be ignored. Levels `01` to `03` SHALL be required and `04` SHALL be optional. The same rule SHALL apply to `.puml` and `.svg` files.

#### Scenario: Four numbered SVG files
- **WHEN** the directory holds `01-context.svg`, `02-container.svg`, `03-component.svg` and `04-code.svg`
- **THEN** the document contains four layers and four navigation buttons in that order

#### Scenario: Optional code level absent
- **WHEN** the directory holds only the `01-`, `02-` and `03-` files
- **THEN** the document contains three layers and three navigation buttons, with no placeholder for the fourth

#### Scenario: Required level absent
- **WHEN** the directory lacks a file for one of `01-`, `02-` or `03-`
- **THEN** the converter reports which level is missing and exits with a non-zero status

#### Scenario: Substring naming no longer recognised
- **WHEN** the directory holds `Context-Diagram.svg` and no file with a numeric prefix
- **THEN** the file is ignored and the converter reports that the context level is missing

### Requirement: PlantUML sources are rendered before stacking
When the input directory contains `.puml` files matching the level prefixes, the converter SHALL render them to SVG with the `plantuml` executable found on `PATH` into a temporary directory, stack the rendered files, and remove the temporary directory before exiting. When `plantuml` is not on `PATH` the converter SHALL report that and exit with a non-zero status.

#### Scenario: Directory of PlantUML sources
- **WHEN** the directory holds `01-context.puml`, `02-container.puml` and `03-component.puml` and `plantuml` is on `PATH`
- **THEN** the document is generated from the rendered SVGs and no rendered files remain in the input directory

#### Scenario: PlantUML unavailable
- **WHEN** the directory holds `.puml` files and `plantuml` is not on `PATH`
- **THEN** the converter reports that `plantuml` was not found and exits with a non-zero status

### Requirement: Each input SVG is embedded as a nested SVG with its own dimensions
For each level the document SHALL contain a group with id `layer-<level>` holding a nested `<svg>` whose `viewBox` is copied from the input file. The input's `width` and `height` SHALL be recorded in a `diagramData` script object keyed by level name, with a `px` suffix stripped. When the input lacks `viewBox`, `width` or `height`, the values `0 0 400 300`, `400` and `300` SHALL be used respectively.

#### Scenario: Dimensions copied
- **WHEN** an input file's root element has `width="800px"`, `height="600px"` and `viewBox="0 0 800 600"`
- **THEN** the nested `<svg>` has `viewBox="0 0 800 600"` and `diagramData` records width 800 and height 600 for that level

#### Scenario: Dimensions absent
- **WHEN** an input file's root element has no `width`, `height` or `viewBox`
- **THEN** the nested `<svg>` has `viewBox="0 0 400 300"` and `diagramData` records width 400 and height 300

### Requirement: Input content is rewritten for the viewer
The converter SHALL remove `<script>` and `<title>` elements, processing instructions and document type declarations from each input. Where a group directly wraps an `<a>` element with an `href`, the `<a>` SHALL be removed and the group SHALL gain `onclick="navigateDown()"` and a pointer cursor. Any other `<a>` elements SHALL be unwrapped, keeping their children. PlantUML note groups SHALL gain the class `note`. A link SHALL gain the class `note-link` when either of its `data-entity-1` or `data-entity-2` attributes equals a note group's id, or, for output from older PlantUML releases, when its id contains the note's id with the `entity_` prefix removed. All other content, including `xlink:href` attributes on `<image>` elements, SHALL be preserved unchanged.

#### Scenario: Linked entity becomes clickable
- **WHEN** an input holds `<g class="entity"><a href="02-container.svg"><rect/></a></g>`
- **THEN** the document holds the group with `onclick="navigateDown()"`, the `<rect/>` inside it, and no `<a>` element

#### Scenario: Scripts and titles removed
- **WHEN** an input holds `<script>` or `<title>` elements
- **THEN** none of them appear in the document

#### Scenario: Note and its link are tagged
- **WHEN** an input holds a note group with id `ent0009` and a link with `data-entity-1="ent0009"`
- **THEN** the note group has class `note` and the link has class `note-link`

#### Scenario: Note and its link are tagged in older PlantUML output
- **WHEN** an input holds a note group with id `entity_GMN49` and a link with id `link_GMN49_loader`
- **THEN** the note group has class `note` and the link has class `note-link`

#### Scenario: Processing instruction removed
- **WHEN** an input holds `<?plantuml 1.2026.8?>` inside the root element
- **THEN** the document does not contain it

#### Scenario: Embedded image preserved
- **WHEN** an input holds `<image xlink:href="data:image/png;base64,...">`
- **THEN** the document holds the same element with the same `xlink:href` value

### Requirement: Output is well-formed and deterministic
The generated document SHALL be well-formed XML. Given the same inputs, the same title and the same timestamp, the converter SHALL produce byte-identical output. The document SHALL carry a `<metadata>` block recording the generator name, version and an RFC 3339 timestamp.

#### Scenario: Golden output
- **WHEN** the converter is run twice on the same directory with the same title and a fixed timestamp
- **THEN** the two outputs are identical and both parse as XML

### Requirement: Malformed input is rejected
When an input file is not well-formed XML or lacks a root `<svg>` element, the converter SHALL report the file name and exit with a non-zero status without writing a document.

#### Scenario: Unclosed element
- **WHEN** an input file contains an unclosed element
- **THEN** the converter names the file in its error and no output file is written
