## Context

See proposal.md for motivation. The current state that shapes the approach:

- `main.go` is one file, one package, with a mutable `SVGStacker` whose methods read and overwrite its fields mid-run.
- Output embeds `time.Now()`, so nothing today can compare output against a checked-in expectation.
- The embedded viewer in `navigation.js` runs `setupLinkHoverEnhancements` on every `showLevel`, inserting nodes and registering a `document` keydown listener each time.
- Two earlier commits fixed XML damage caused by regex rewriting and by `encoding/xml` re-encoding `xlink` without a namespace declaration in scope. Both are constraints on the single-parser design.
- `getBBox` returns zeros for elements inside a `display:none` ancestor, so label hitboxes cannot be measured before a layer is first shown.

## Goals / Non-Goals

**Goals:**
- Every step leaves the test suite green and the generated `examples/example.svg` regenerable.
- The converter is a composition of pure functions over a `[]Diagram` sequence, with I/O confined to `main`.
- The converter-to-viewer seam is a small, named set of classes and IDs, listed once in the design and asserted by the golden test.

**Non-Goals:**
- Full viewer test coverage. This change adds the harness and tests for two requirements, initial state and enhancement-once, to establish the pattern. The remaining viewer requirements are covered in a later change using that pattern.
- The Claude Code skill that replaces the `prompt` subcommand. This change removes the subcommand and preserves its prompt as a document the skill can be built from.
- Any change to viewer features or appearance.

## Decisions

### Order of work: safety net first, then subtraction, then restructuring, then the viewer

The steps run in this order, each committed separately: deterministic output and golden test; remove `prompt`; remove dead code; level table; single parser; template and pipeline; `flag` CLI; viewer split; documentation. Subtraction comes before restructuring because there is less to restructure afterwards. The viewer comes last because its split depends on the classes the new parser emits.

Alternative considered: one rewrite commit. Rejected because a golden-file diff per step is what makes each step reviewable.

### Timestamp is a parameter, not dropped

`stack` takes the timestamp as an argument and `main` passes `time.Now()`. Tests pass a fixed value. The metadata block is a released feature since 0.6.0 and costs nothing to keep once it is injectable. Removing it would also be a spec change for no gain.

### Levels are an ordered slice of `{prefix, name}` structs

The four levels are a sequence, not a set or map, because order is the thing every consumer needs: discovery sorts by it, buttons and layers render in it, and `availableLevels` is it filtered to what was found. A package-level `var levels = []level{{"01","context"},...}` with a comment at the declaration site. Discovery matches `^<prefix>-` on the base name for both `.puml` and `.svg`. The substring matcher goes, which is why the spec marks `.svg` discovery as breaking.

### Diagrams are a `[]Diagram`, never a map

The current `map[string]DiagramInfo` forces every consumer to re-walk the `levels` slice to recover order, and would make output non-deterministic if it were ever ranged over directly. A slice in level order removes both problems.

### One tokeniser pass with a token slice for lookahead

Each input is decoded with `encoding/xml` into a `[]xml.Token` (each copied with `xml.CopyToken`, since the decoder reuses buffers). Working on a slice rather than a stream gives the one piece of lookahead the rewrite needs: when a `g` start element is followed by an `a` start element with `href`, the `g` gains the `onclick` attribute and the `a` start and its matching end are dropped. Everything else is a filter over the slice: drop `script` and `title` subtrees, unwrap remaining `a` elements, add `class="note"` when a `g.entity` subtree contains a `path` with fill `#FEFFDD`, add `class="note-link"` to `g.link` elements attached to a tagged note. PlantUML 1.2026 names links by `data-entity-1` and `data-entity-2` attributes equal to the target group's `id`; older releases encoded the relationship in the link id (`link_<note>_<target>`). Both forms are recognised because checked-in examples were rendered with the older release and the machine rendering them now has the newer one. Processing instructions such as `<?plantuml 1.2026.8?>` are dropped as well. Dimensions are read from the root `svg` start element's attributes in the same pass. The output is re-encoded with indentation.

The known `xlink` mangling is handled the same way the current code handles it: the re-encoded tokens are wrapped in a root element that declares `xmlns:xlink`, and the wrapper's start and end tokens are skipped on output. The golden test's example input must include an `<image xlink:href>` so this stays covered.

Alternatives considered: keep the regex parser for dimensions and use `encoding/xml` only for pretty-printing (status quo, the source of two past bugs); a streaming transform without a slice (loses the lookahead and needs a small state machine for no benefit at these sizes).

### Note detection moves to the converter, viewer reads classes

The viewer currently identifies notes by fill colour and links by id substring at runtime. Both are properties of the PlantUML output, not of the reader's interaction, so the converter computes them once and the viewer toggles on the class. The seam contract, in full:

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

The golden test asserts this table by construction. Nothing else in the document is relied on by the script.

### Template rendering with `text/template`

The document is one template with the CSS, header, buttons, layers and script. `diagramData` and `availableLevels` are serialised with `encoding/json` rather than hand-built with `fmt.Sprintf`, which also removes the trailing-comma bookkeeping. Diagram bodies are inserted unescaped, as they are already XML. The JS source is a second embedded file included by the template.

### `flag` package with a positional directory

`flag.NewFlagSet` with `--output`, `--title` and `--version`; `-h` and `--help` come free. `flag.Arg(0)` is the directory. The sentinel-error dispatch in `parseArgsSlice` is deleted along with the `prompt` branch.

### Viewer enhancement runs on first show, tracked per layer

Because `getBBox` needs a rendered layer, hitboxes cannot be created at load for hidden layers. The viewer keeps a `Set` of enhanced level names; `showLevel` makes the layer visible, then enhances it only if its name is not in the set. The Escape listener is registered once at the top level of the script. Everything that does not need layout, which is title removal and note detection, has moved to the converter, so the enhancement function shrinks to hitbox and background creation plus the hover and click handlers.

### Viewer tests run the generated document in headless Chromium

The viewer is a script over a known document shape, and its requirements are stated in terms of DOM state after interactions, so the test has to run a real DOM with real layout: `getBBox` is central to enhancement and no pure-JS DOM stub implements it. Headless Chromium provides that with no Go dependency. A Go test locates `chromium`, `chromium-browser`, `google-chrome` or `google-chrome-stable` on `PATH` and skips with a message when none is found, so `go test ./...` stays green on machines without a browser.

The mechanism: the test generates a document with `stack`, strips the XML declaration, and writes it inline into an HTML harness alongside a `<pre id="results">` and a driver `<script>`. The driver calls the viewer's functions (`showLevel`, `toggleNotes`, click handlers via `dispatchEvent`), then writes a JSON object of observations into the `<pre>`. Chromium is run with `--headless=new --dump-dom --virtual-time-budget=2000`, which runs scripts and timers to completion before dumping, and the test parses the JSON out of the dump with `encoding/json`. Each viewer test is one driver script and one set of assertions on the parsed object. This was verified to capture both a `setTimeout` callback and a non-zero `getBBox` before being adopted.

The first two tests written under this pattern: initial state (only the first layer visible, its button active), and enhancement-once (after ten switches between two levels, every link on both layers has exactly one `.label-hitbox` and one `.text-bg`, and switching does not grow the count). The second is the defect this change fixes, so it is written to fail before the viewer split and pass after.

Alternatives considered: `chromedp` (a Go dependency, and the project has none); Node with `jsdom` (no layout, so `getBBox` returns zeros and enhancement cannot be exercised); Playwright (a Node dependency and its own browser download).

### The `prompt` subcommand's prompt is preserved as a document

Before deletion, the exact text `runPromptCommand` assembles is written to `docs/c4-diagram-prompt.md`: the preamble, the instruction to save into `docs/c4/`, a line marking where `C4-DIAGRAM-SPEC.md` is inserted in full, the project-context section with each gathered value replaced by a named placeholder and a note on how it was derived, the closing file-list instruction, and the follow-up command that stacks `docs/c4/`. The spec file is referenced rather than copied so it has one source. The document is the raw material for the future skill and nothing in the binary reads it.

### Golden fixtures live in `testdata/<case>/`

Each case is a directory of numbered input SVGs plus `expected.svg`. Inputs are hand-written and small, but one case must include an `<image xlink:href>` and one must include a note with an attached link, so the two historical bug classes stay pinned. The test runs `stack` with a fixed timestamp and compares byte-for-byte. A `-update` flag rewrites `expected.svg` for intentional output changes, and the resulting diff is reviewed in the commit.

## Risks / Trade-offs

- [Golden files churn on every intentional output change] → each step's commit includes the regenerated `expected.svg`, so the diff is the review artefact rather than noise.
- [`.svg` directories named by substring stop working] → documented as breaking in the spec; the README already states the numeric convention, so only undocumented usage is affected.
- [The token-slice rewrite mishandles a PlantUML construct not present in fixtures] → `examples/` is regenerated with real PlantUML output at the end of every step and diffed by eye once, then the relevant construct is added to a fixture.
- [`encoding/xml` re-encodes attribute order or self-closing tags differently from the input] → acceptable; the output is for browsers, and the golden test pins whatever the encoder produces.
- [Removing `prompt` breaks a user's workflow] → the changelog marks it breaking and points at `docs/c4-diagram-prompt.md`, which can be pasted into Claude Code by hand until the skill exists.
- [Chromium output format or flags change across versions] → the test depends only on `--dump-dom` emitting the final DOM and on the JSON inside one `<pre>`; the driver writes nothing else, and a parse failure reports the raw dump.
- [Viewer tests are skipped in an environment without Chromium and a regression goes unnoticed] → the skip prints a visible message, and the environment that regenerates `examples/example.svg` has Chromium.
