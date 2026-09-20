package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
)

//go:embed navigation.js
var navigationJS string

//go:embed stacked.svg.tmpl
var document_template_source string

var version = "unreleased"

// converts a string to title case (first letter uppercase, rest as-is).
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// one C4 level: the filename prefix that identifies it and whether a diagram set must include it
type level struct {
	prefix   string
	name     string
	required bool
}

// the C4 levels in drill-down order. An ordered slice, not a map, because every consumer
// (file discovery, button order, layer order, the viewer's level list) needs the order
// and derives it from this one table.
var levels = []level{
	{"01", "context", true},
	{"02", "container", true},
	{"03", "component", true},
	{"04", "code", false},
}

// a level paired with the input file found for it
type level_file struct {
	level level
	path  string
}

// finds one file per level in `dir` with extension `ext` (including the dot), matched by
// `<prefix>-` at the start of the base name. Returns levels in table order, omitting an
// absent optional level and failing on an absent required one.
func discover_levels(dir, ext string) ([]level_file, error) {
	var found []level_file
	for _, lvl := range levels {
		matches, err := filepath.Glob(filepath.Join(dir, lvl.prefix+"-*"+ext))
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			if lvl.required {
				return nil, fmt.Errorf("missing required %s level: no %s-*%s file in %s", lvl.name, lvl.prefix, ext, dir)
			}
			continue
		}
		sort.Strings(matches)
		found = append(found, level_file{lvl, matches[0]})
	}
	return found, nil
}

// one parsed input diagram: its level, the dimensions the viewer needs, and the rewritten body
type Diagram struct {
	level   level
	viewBox string
	width   float64
	height  float64
	body    string
}

// the SVG namespace; elements in it are emitted without a namespace so they inherit the outer document's
const svg_namespace = "http://www.w3.org/2000/svg"

// fill colour PlantUML gives note bodies; the only stable marker that a group is a note
const note_fill = "#FEFFDD"

// decodes an XML document into a token slice. A slice rather than a stream so the rewrite
// can look ahead (a group followed by a link) and back (a link naming an earlier note).
func decode_tokens(src []byte) ([]xml.Token, error) {
	decoder := xml.NewDecoder(bytes.NewReader(src))
	decoder.Entity = xml.HTMLEntity
	var tokens []xml.Token
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return tokens, nil
		}
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, xml.CopyToken(token))
	}
}

// index of the end element closing the start element at `i`
func subtree_end(tokens []xml.Token, i int) int {
	depth := 0
	for j := i; j < len(tokens); j++ {
		switch tokens[j].(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return len(tokens) - 1
}

// index of the next token at `i` or later that is not whitespace or a comment
func next_significant(tokens []xml.Token, i int) int {
	for ; i < len(tokens); i++ {
		switch t := tokens[i].(type) {
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return i
			}
		case xml.Comment:
		default:
			return i
		}
	}
	return len(tokens)
}

func attr(el xml.StartElement, local string) string {
	for _, a := range el.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func set_attr(el xml.StartElement, local, value string) xml.StartElement {
	for i, a := range el.Attr {
		if a.Name.Local == local {
			el.Attr[i].Value = value
			return el
		}
	}
	el.Attr = append(el.Attr, xml.Attr{Name: xml.Name{Local: local}, Value: value})
	return el
}

func has_class(el xml.StartElement, class string) bool {
	for _, c := range strings.Fields(attr(el, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

func add_class(el xml.StartElement, class string) xml.StartElement {
	if has_class(el, class) {
		return el
	}
	return set_attr(el, "class", strings.TrimSpace(attr(el, "class")+" "+class))
}

// parses a length such as "494px" or "494", falling back to `fallback`
func parse_length(value string, fallback float64) float64 {
	n, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(value), "px"), 64)
	if err != nil {
		return fallback
	}
	return n
}

// removes tokens the viewer must not receive: script and title subtrees, processing
// instructions, directives, and whitespace between elements
func strip_tokens(tokens []xml.Token) []xml.Token {
	var out []xml.Token
	for i := 0; i < len(tokens); i++ {
		switch t := tokens[i].(type) {
		case xml.StartElement:
			if t.Name.Local == "script" || t.Name.Local == "title" {
				i = subtree_end(tokens, i)
				continue
			}
		case xml.ProcInst, xml.Directive:
			continue
		case xml.CharData:
			if strings.TrimSpace(string(t)) == "" {
				continue
			}
		}
		out = append(out, tokens[i])
	}
	return out
}

// turns a group that directly wraps a linked `<a>` into a clickable drill-down and unwraps every other `<a>`
func rewrite_links(tokens []xml.Token) []xml.Token {
	drop := map[int]bool{}
	for i, token := range tokens {
		g, ok := token.(xml.StartElement)
		if !ok || g.Name.Local != "g" {
			continue
		}
		j := next_significant(tokens, i+1)
		if j >= len(tokens) {
			break
		}
		a, ok := tokens[j].(xml.StartElement)
		if !ok || a.Name.Local != "a" || attr(a, "href") == "" {
			continue
		}
		g = set_attr(g, "onclick", "navigateDown()")
		g = set_attr(g, "style", strings.TrimSpace(attr(g, "style")+" cursor:pointer;"))
		tokens[i] = g
		drop[j] = true
		drop[subtree_end(tokens, j)] = true
	}
	var out []xml.Token
	for i, token := range tokens {
		if drop[i] {
			continue
		}
		if el, ok := token.(xml.StartElement); ok && el.Name.Local == "a" {
			drop[subtree_end(tokens, i)] = true
			continue
		}
		out = append(out, token)
	}
	return out
}

// tags note groups with class `note` and the links attached to them with class `note-link`.
// A link is attached when a `data-entity-*` attribute names the note's id (current PlantUML)
// or when the link's id contains the note's `entity_` suffix (older PlantUML).
func tag_notes(tokens []xml.Token) []xml.Token {
	var note_ids []string
	for i, token := range tokens {
		g, ok := token.(xml.StartElement)
		if !ok || g.Name.Local != "g" || !has_class(g, "entity") {
			continue
		}
		end := subtree_end(tokens, i)
		for _, inner := range tokens[i+1 : end] {
			if p, ok := inner.(xml.StartElement); ok && p.Name.Local == "path" && strings.EqualFold(attr(p, "fill"), note_fill) {
				tokens[i] = add_class(g, "note")
				note_ids = append(note_ids, attr(g, "id"))
				break
			}
		}
	}
	for i, token := range tokens {
		g, ok := token.(xml.StartElement)
		if !ok || g.Name.Local != "g" || !has_class(g, "link") {
			continue
		}
		for _, id := range note_ids {
			if id == "" {
				continue
			}
			suffix := strings.TrimPrefix(id, "entity_")
			attached := attr(g, "data-entity-1") == id || attr(g, "data-entity-2") == id ||
				(suffix != id && strings.Contains(attr(g, "id"), suffix))
			if attached {
				tokens[i] = add_class(g, "note-link")
				break
			}
		}
	}
	return tokens
}

// re-encodes tokens as indented XML. Elements in the SVG namespace lose their namespace so
// the encoder does not redeclare `xmlns` on each one; namespace declarations are dropped
// and the encoder re-declares prefixes such as `xlink` where they are used.
func encode_tokens(tokens []xml.Token) (string, error) {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("      ", "  ")
	for _, token := range tokens {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Space == svg_namespace {
				t.Name.Space = ""
			}
			var attrs []xml.Attr
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
					continue
				}
				attrs = append(attrs, a)
			}
			t.Attr = attrs
			token = t
		case xml.EndElement:
			if t.Name.Space == svg_namespace {
				t.Name.Space = ""
			}
			token = t
		}
		if err := encoder.EncodeToken(token); err != nil {
			return "", err
		}
	}
	if err := encoder.Flush(); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// parses one SVG document for `lvl` into its dimensions and a rewritten body ready for embedding
func parse_diagram(lvl level, src []byte) (Diagram, error) {
	diagram := Diagram{level: lvl}
	tokens, err := decode_tokens(src)
	if err != nil {
		return diagram, err
	}
	start := next_significant(tokens, 0)
	for start < len(tokens) {
		if _, ok := tokens[start].(xml.StartElement); ok {
			break
		}
		start = next_significant(tokens, start+1)
	}
	if start >= len(tokens) {
		return diagram, fmt.Errorf("no root <svg> element")
	}
	root := tokens[start].(xml.StartElement)
	if root.Name.Local != "svg" {
		return diagram, fmt.Errorf("root element is <%s>, not <svg>", root.Name.Local)
	}
	end := subtree_end(tokens, start)

	diagram.viewBox = attr(root, "viewBox")
	if diagram.viewBox == "" {
		diagram.viewBox = "0 0 400 300"
	}
	diagram.width = parse_length(attr(root, "width"), 400)
	diagram.height = parse_length(attr(root, "height"), 300)

	body := tag_notes(rewrite_links(strip_tokens(tokens[start+1 : end])))
	diagram.body, err = encode_tokens(body)
	return diagram, err
}

// reads and parses the numbered SVG files in `dir`, in level order
func load(dir string) ([]Diagram, error) {
	found, err := discover_levels(dir, ".svg")
	if err != nil {
		return nil, err
	}
	var diagrams []Diagram
	for _, f := range found {
		src, err := os.ReadFile(f.path)
		if err != nil {
			return nil, err
		}
		diagram, err := parse_diagram(f.level, src)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.path, err)
		}
		diagrams = append(diagrams, diagram)
	}
	return diagrams, nil
}

// renders the numbered `.puml` files in `dir` to SVG in a temporary directory. The caller
// runs `cleanup` to remove it.
func render(dir string) (svg_dir string, cleanup func(), err error) {
	found, err := discover_levels(dir, ".puml")
	if err != nil {
		return "", nil, err
	}
	plantuml, err := exec.LookPath("plantuml")
	if err != nil {
		return "", nil, fmt.Errorf("plantuml not found in PATH: %w", err)
	}
	svg_dir, err = os.MkdirTemp("", "svg-stacker-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(svg_dir) }

	args := []string{"-tsvg", "-o", svg_dir, "-nbthread", "auto"}
	for _, f := range found {
		args = append(args, f.path)
	}
	output, err := exec.Command(plantuml, args...).CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("plantuml failed: %w\n%s", err, output)
	}
	return svg_dir, cleanup, nil
}

// the document template. Everything static about the output lives in the template file; Go
// supplies only the per-level values and the two JSON blocks the viewer reads.
var document_template = template.Must(template.New("stacked").Parse(document_template_source))

// one level as the template renders it
type layer_view struct {
	Name    string
	Label   string
	ViewBox string
	Body    string
	ButtonX int
	TextX   int
}

type document_view struct {
	Title           string
	Version         string
	Timestamp       string
	Layers          []layer_view
	DiagramData     string
	AvailableLevels string
	Script          string
}

// the per-level dimensions the viewer reads from `diagramData`
type diagram_dimensions struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Ratio  float64 `json:"ratio"`
}

// renders the stacked document for `diagrams`, which must be in level order
func stack(title string, at time.Time, diagrams []Diagram) (string, error) {
	view := document_view{
		Title:     title,
		Version:   version,
		Timestamp: at.UTC().Format(time.RFC3339),
		Script:    navigationJS,
	}
	dimensions := map[string]diagram_dimensions{}
	names := []string{}
	for i, d := range diagrams {
		x := 26 + i*117
		view.Layers = append(view.Layers, layer_view{
			Name: d.level.name, Label: titleCase(d.level.name), ViewBox: d.viewBox, Body: d.body,
			ButtonX: x, TextX: x + 13,
		})
		dimensions[d.level.name] = diagram_dimensions{
			Width: math.Round(d.width), Height: math.Round(d.height), Ratio: math.Round(d.width/d.height*100) / 100,
		}
		names = append(names, d.level.name)
	}
	dimensions_json, err := json.MarshalIndent(dimensions, "    ", "  ")
	if err != nil {
		return "", err
	}
	names_json, err := json.Marshal(names)
	if err != nil {
		return "", err
	}
	view.DiagramData = string(dimensions_json)
	view.AvailableLevels = string(names_json)

	var buf bytes.Buffer
	if err := document_template.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// converts the diagrams in `input_dir` and writes the document to `output_file`, or to
// standard output when it is empty
func run(input_dir, output_file, title string) error {
	if title == "" {
		title = "🏗️ Stacked C4 Architecture"
	}
	dir := input_dir
	if pumls, _ := filepath.Glob(filepath.Join(input_dir, "*.puml")); len(pumls) > 0 {
		svg_dir, cleanup, err := render(input_dir)
		if err != nil {
			return err
		}
		defer cleanup()
		dir = svg_dir
	}
	diagrams, err := load(dir)
	if err != nil {
		return err
	}
	document, err := stack(title, time.Now(), diagrams)
	if err != nil {
		return err
	}
	if output_file == "" {
		_, err = fmt.Print(document)
		return err
	}
	return os.WriteFile(output_file, []byte(document), 0644)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: svg-stacker <directory> [OPTIONS]

Combines the numbered SVG or PlantUML files in <directory> into one stacked SVG.

OPTIONS:
  -h, --help          Show this help message and exit
  -v, --version       Show version information and exit
  --output FILE       Output file path (default: stdout)
  --title TITLE       Title for the diagram (default: "🏗️ Stacked C4 Architecture")

EXAMPLES:
  svg-stacker ./examples
  svg-stacker ./examples --output output.svg
  svg-stacker ./examples --title "My Architecture"
`)
}

func printVersion() {
	fmt.Printf("svg-stacker version %s\n", version)
}

func parseArgsSlice(args []string) (inputDir, outputFile, title string, err error) {
	if len(args) < 1 {
		return "", "", "", fmt.Errorf("directory argument required")
	}

	// Check for help/version flags first
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return "", "", "", fmt.Errorf("help")
		}
		if arg == "-v" || arg == "--version" {
			return "", "", "", fmt.Errorf("version")
		}
	}

	inputDir = args[0]
	outputFile = ""
	title = ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			} else {
				return "", "", "", fmt.Errorf("--output requires an argument")
			}
		case "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			} else {
				return "", "", "", fmt.Errorf("--title requires an argument")
			}
		case "-h", "--help", "-v", "--version":
			// Already handled above
		default:
			// Unknown flag
			return "", "", "", fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	return inputDir, outputFile, title, nil
}

func parseArgs() (inputDir, outputFile, title string, shouldExit bool, exitCode int) {
	if len(os.Args) < 2 {
		printUsage()
		return "", "", "", true, 1
	}

	inputDir, outputFile, title, err := parseArgsSlice(os.Args[1:])
	if err == nil {
		return inputDir, outputFile, title, false, 0
	}

	// Handle special cases
	switch err.Error() {
	case "help":
		printUsage()
		return "", "", "", true, 0
	case "version":
		printVersion()
		return "", "", "", true, 0
	default:
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Use 'svg-stacker --help' for usage information\n")
		return "", "", "", true, 1
	}
}

func main() {
	inputDir, outputFile, title, shouldExit, exitCode := parseArgs()
	if shouldExit {
		os.Exit(exitCode)
	}
	if err := run(inputDir, outputFile, title); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
