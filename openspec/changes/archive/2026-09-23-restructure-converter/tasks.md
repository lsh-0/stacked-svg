## 1. Safety net

- [x] 1.1 Make the timestamp a parameter of document generation, with `main` passing `time.Now()`, and verify `go build` succeeds and `examples/example.svg` regenerates with only the timestamp line differing
- [x] 1.2 Create `testdata/basic/` with three hand-written numbered input SVGs and `testdata/notes-and-images/` with four inputs including a note group, a link referencing its entity id, a linked `<a>` wrapper, a `<script>`, a `<title>` and an `<image xlink:href>`; verify each input parses with `xmllint --noout` or `encoding/xml`
- [x] 1.3 Write a golden test that runs generation over each `testdata/<case>/` with a fixed timestamp and compares to `expected.svg`, with an `-update` flag to rewrite expectations; generate the initial expectations and verify `go test ./...` passes and the expectations parse as XML
- [x] 1.4 Delete `copyTestdataFiles`, the old `testdata/*.svg` fixtures, and the duplicate `TestActualGeneratedFiles`; verify `go test ./...` passes and `go vet ./...` is clean
- [x] 1.5 Commit

## 2. Remove the Claude integration

- [x] 2.1 Write `docs/c4-diagram-prompt.md` holding the exact prompt `runPromptCommand` assembles, with the spec referenced by file name, the project-context values as named placeholders with a note on how each was derived, and the follow-up stacking command; verify by diffing it against the output of a temporary `go run` that prints the assembled prompt for this repository
- [x] 2.2 Delete `runPromptCommand`, `gatherProjectContext`, `ProjectContext`, the `C4-DIAGRAM-SPEC.md` embed, the `prompt` branch in argument parsing and its usage text; verify `go build` succeeds and `./svg-stacker prompt` reports a missing directory
- [x] 2.3 Replace the "Generating C4 Diagrams with Claude" section in `README.md` with a short pointer to `docs/c4-diagram-prompt.md`, and remove the Claude references from the usage text; verify `grep -i claude main.go` is empty
- [x] 2.4 Verify the golden test output is unchanged and commit

## 3. Remove dead code

- [x] 3.1 Delete the "diagram not found" placeholder branch in layer generation; verify the golden test is unchanged and the three-level case has no fourth layer
- [x] 3.2 Commit

## 4. Single level table

- [x] 4.1 Declare an ordered `levels` slice of prefix and name pairs with a justification comment, and rewrite `.puml` and `.svg` discovery to match `^<prefix>-` on the base name for both; verify a new unit test covers a four-file directory, a three-file directory, a missing required level and a substring-named file that is ignored
- [x] 4.2 Derive button order, layer order and `availableLevels` from the table and remove the inline `levels` literal and `extractLevel`; verify the golden test is unchanged and `TestExtractLevel` is replaced by the discovery test
- [x] 4.3 Commit

## 5. Single parser

- [x] 5.1 Write a `parseDiagram` function that decodes an input into a copied `[]xml.Token`, reads `viewBox`, `width` and `height` from the root `svg` start element with the documented defaults, and returns a `Diagram` value; verify unit tests cover present dimensions, a `px` suffix, absent dimensions, a missing root `svg`, and malformed XML naming the file
- [x] 5.2 Add token-slice transforms that drop `script` and `title` subtrees, rewrite a `g` directly wrapping an `a[href]` into `onclick="navigateDown()"` with a pointer cursor, and unwrap any other `a`; verify unit tests cover each transform and that `xlink:href` on `image` survives re-encoding
- [x] 5.3 Add transforms that set `class="note"` on `g.entity` subtrees containing a `path` with fill `#FEFFDD` and `class="note-link"` on `g.link` elements whose id contains a tagged note's entity id; verify a unit test covers a note with one attached and one unattached link
- [x] 5.4 Replace `parseSVG`, `cleanDiagramContent`, `prettyPrintXML` and `ValidateXML` with the new parser and re-encoder; regenerate the golden expectations, review the diff for the new classes and unchanged content, and verify `go test ./...` passes
- [x] 5.5 Commit

## 6. Pure pipeline and template

- [x] 6.1 Introduce `stack(title string, at time.Time, diagrams []Diagram) string` rendering one `text/template` with `diagramData` and `availableLevels` serialised by `encoding/json`, and move the CSS, header, buttons, layers and script into the template; verify the golden expectations change only in JSON formatting and `go test ./...` passes after `-update`
- [x] 6.2 Extract PlantUML rendering into `render(dir string) (svgDir string, cleanup func(), err error)` and delete `SVGStacker`, `NewSVGStacker`, `CreateStackedSVG` and `DiagramInfo`; verify `main` is the only function performing I/O and `go vet ./...` is clean
- [x] 6.3 Reorder `main.go` so declarations with fewer in-file dependencies come first and `main` is last; verify `gofmt -l` is empty and `go build` succeeds
- [x] 6.4 Regenerate `examples/example.svg` from `examples/` with real PlantUML and verify it opens in a browser with all four levels, drill-down, notes toggle and path hover working
- [x] 6.5 Commit

## 7. Standard flag parsing

- [x] 7.1 Replace `parseArgsSlice`, `parseArgs`, `printUsage` and the sentinel-error dispatch with a `flag.FlagSet` defining `--output`, `--title` and `--version`, taking `flag.Arg(0)` as the directory; verify `TestParseArgsSlice` is rewritten against the new function and covers directory only, both flags, unknown flag, missing directory and `--version`
- [x] 7.2 Verify `./svg-stacker -h`, `./svg-stacker --version`, `./svg-stacker examples/ --output out.svg --title "T"` and `./svg-stacker --bogus` behave per the spec, then commit

## 8. Viewer split

- [x] 8.1 Write `viewer_test.go` with a helper that finds a Chromium binary on `PATH` or skips, wraps a `stack` document in an HTML harness with a driver script and a results `<pre>`, runs `--headless=new --dump-dom --virtual-time-budget=2000`, and parses the JSON results; verify with a first test asserting the initial state: only the first layer is visible and its button is active, and `go test -run Viewer -v` shows it passing and shows the skip message when `PATH` is emptied
- [x] 8.2 Add the enhancement-once viewer test: switch between two levels ten times and record the count of `.label-hitbox` and `.text-bg` per link on both layers; verify it fails against the current script with counts greater than one
- [x] 8.3 Rewrite `toggleNotes` to toggle `.note` and `.note-link` by class on every layer and delete the fill-colour and id-substring detection; verify in a browser on `examples/example.svg` that notes and their links hide and show together
- [x] 8.4 Delete the `<title>` removal from `setupLinkHoverEnhancements` and move the Escape `keydown` listener to a single top-level registration; verify in the browser that Escape still clears pins and the DevTools listener panel shows one keydown listener on `document`
- [x] 8.5 Track enhanced levels in a `Set` and enhance a layer only on its first `showLevel`; verify the enhancement-once test from 8.2 now passes
- [x] 8.6 Regenerate `examples/example.svg` and the golden expectations, verify `go test ./...` passes, and commit

## 9. Documentation

- [x] 9.1 Rewrite `CLAUDE.md` to describe the current build, test and generate commands, the file layout, the level convention and the converter-to-viewer seam table from the design; verify no reference to Node, `output/`, a hard-coded PlantUML path or the `prompt` command remains
- [x] 9.2 Update `README.md`: project structure list, usage examples, and a note under file naming that `.svg` inputs follow the same numeric convention; verify `grep -n prompt README.md` is empty
- [x] 9.3 Remove the satisfied "Replace the Claude integration with a skill" entry from `TODO.md` or narrow it to the skill alone; verify the file still parses as `---`-delimited entries
- [x] 9.4 Commit
