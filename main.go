package main

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

//go:embed navigation.js
var navigationJS string

type SVGStacker struct {
	diagrams   map[string]DiagramInfo
	inputDir   string
	outputFile string
	title      string
	tempDir    string
	now        time.Time // generation time written to the metadata block; injected so output is reproducible in tests
}

type DiagramInfo struct {
	content     string
	viewBox     string
	width       float64
	height      float64
	aspectRatio float64
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

	stacker := NewSVGStacker(inputDir, outputFile, title)
	if err := stacker.CreateStackedSVG(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewSVGStacker(inputDir, outputFile, title string) *SVGStacker {
	if title == "" {
		title = "🏗️ Stacked C4 Architecture"
	}
	return &SVGStacker{
		diagrams:   make(map[string]DiagramInfo),
		inputDir:   inputDir,
		outputFile: outputFile,
		title:      title,
		now:        time.Now(),
	}
}

func (s *SVGStacker) CreateStackedSVG() error {
	// Check if input directory contains .puml files
	hasPuml, err := s.hasPumlFiles()
	if err != nil {
		return err
	}

	if hasPuml {
		// Generate SVG files from PlantUML
		if err := s.generateSVGsFromPuml(); err != nil {
			return err
		}
		// Clean up temp directory on exit
		defer func() {
			if s.tempDir != "" {
				os.RemoveAll(s.tempDir)
			}
		}()
	}

	// Load all SVG files
	if err := s.loadDiagrams(); err != nil {
		return err
	}

	// Create the master SVG
	stackedSVG := s.buildStackedSVG()

	// Write to stdout or file
	if s.outputFile == "" {
		fmt.Print(stackedSVG)
	} else {
		if err := os.WriteFile(s.outputFile, []byte(stackedSVG), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (s *SVGStacker) hasPumlFiles() (bool, error) {
	files, err := filepath.Glob(filepath.Join(s.inputDir, "*.puml"))
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

func (s *SVGStacker) generateSVGsFromPuml() error {
	found, err := discover_levels(s.inputDir, ".puml")
	if err != nil {
		return err
	}
	var pumlFiles []string
	for _, f := range found {
		pumlFiles = append(pumlFiles, f.path)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "svg-stacker-*")
	if err != nil {
		return err
	}
	s.tempDir = tempDir

	// Run plantuml to generate SVG files
	plantumlPath, err := exec.LookPath("plantuml")
	if err != nil {
		return fmt.Errorf("plantuml not found in PATH: %w", err)
	}

	args := []string{"-tsvg", "-o", tempDir, "-nbthread", "auto"}
	args = append(args, pumlFiles...)

	cmd := exec.Command(plantumlPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "PlantUML output: %s\n", string(output))
		return fmt.Errorf("plantuml failed: %w", err)
	}

	// Update inputDir to point to temp directory
	s.inputDir = tempDir
	return nil
}

func (s *SVGStacker) loadDiagrams() error {
	found, err := discover_levels(s.inputDir, ".svg")
	if err != nil {
		return err
	}

	for _, f := range found {
		content, err := os.ReadFile(f.path)
		if err != nil {
			return err
		}

		info, err := parse_diagram(content)
		if err != nil {
			return fmt.Errorf("%s: %w", f.path, err)
		}

		s.diagrams[f.level.name] = info
	}

	return nil
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

// parses one SVG document into its dimensions and a rewritten body ready for embedding
func parse_diagram(src []byte) (DiagramInfo, error) {
	var info DiagramInfo
	tokens, err := decode_tokens(src)
	if err != nil {
		return info, err
	}
	start := next_significant(tokens, 0)
	for start < len(tokens) {
		if _, ok := tokens[start].(xml.StartElement); ok {
			break
		}
		start = next_significant(tokens, start+1)
	}
	if start >= len(tokens) {
		return info, fmt.Errorf("no root <svg> element")
	}
	root := tokens[start].(xml.StartElement)
	if root.Name.Local != "svg" {
		return info, fmt.Errorf("root element is <%s>, not <svg>", root.Name.Local)
	}
	end := subtree_end(tokens, start)

	info.viewBox = attr(root, "viewBox")
	if info.viewBox == "" {
		info.viewBox = "0 0 400 300"
	}
	info.width = parse_length(attr(root, "width"), 400)
	info.height = parse_length(attr(root, "height"), 300)
	info.aspectRatio = info.width / info.height

	body := tag_notes(rewrite_links(strip_tokens(tokens[start+1 : end])))
	info.content, err = encode_tokens(body)
	return info, err
}

func (s *SVGStacker) buildStackedSVG() string {
	// Use embedded JavaScript for interactive mode
	jsContent := []byte(navigationJS)

	var sb strings.Builder

	// SVG Header - JavaScript will set explicit dimensions
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg"
     xmlns:xlink="http://www.w3.org/1999/xlink"
     width="1920"
     height="1080"
     style="background: #f8f9fa; display: block;">

  <title>Stacked C4 Architecture Diagrams</title>

  <!-- Generator Metadata (invisible) -->
  <metadata>
    <generator>stacked-c4-svg</generator>
    <version>` + version + `</version>
    <timestamp>` + s.now.UTC().Format(time.RFC3339) + `</timestamp>
  </metadata>

  <!-- CSS Styles for Progressive Enhancement -->
  <style>
    /* Path highlighting - works without JavaScript */
    .link path,
    .link polygon {
      pointer-events: stroke; /* Only capture events on the stroke itself */
    }

    /* Highlight when JavaScript adds highlighted class (triggered by text hover) */
    .link.highlighted path,
    .link.highlighted polygon {
      stroke: #e74c3c !important;
      stroke-width: 3 !important;
      filter: drop-shadow(0 0 3px rgba(231, 76, 60, 0.5));
    }

    /* Make link text labels hoverable and disable tooltips */
    .link text {
      cursor: pointer;
      user-select: none;
      pointer-events: all;
    }

    /* Make text white when link is highlighted so it shows on red background */
    .link.highlighted text {
      fill: white !important;
    }

    /* Hide any title elements that might trigger tooltips */
    .link title {
      display: none;
    }

    /* Dimmed state (applied by JavaScript) */
    .link.dimmed path,
    .link.dimmed polygon {
      opacity: 0.3;
    }
  </style>`)

	sb.WriteString(fmt.Sprintf(`

  <!-- Navigation Header -->
  <rect x="0" y="0" width="100%%" height="80" fill="#2c3e50"/>
  <text x="26" y="50" font-family="Arial, sans-serif" font-size="30" font-weight="bold" fill="white">
    %s
  </text>

  <!-- Navigation Buttons -->
`, s.title))

	// Generate navigation buttons (only for levels that exist)
	buttonIndex := 0
	for _, lvl := range levels {
		level := lvl.name
		if _, exists := s.diagrams[level]; !exists {
			continue // Skip button if diagram doesn't exist
		}

		x := 26 + buttonIndex*117
		buttonIndex++

		sb.WriteString(fmt.Sprintf(`  <rect x="%d" y="91" width="104" height="33" rx="4"
        fill="#3498db" stroke="#2980b9" stroke-width="1"
        style="cursor:pointer" onclick="showLevel('%s')"
        id="nav-%s"/>
  <text x="%d" y="113" font-family="Arial, sans-serif" font-size="14"
        fill="white" style="cursor:pointer; user-select: none"
        onclick="showLevel('%s')">
    %s
  </text>
`, x, level, level, x+13, level, titleCase(level)))
	}

	// Add toggle buttons (positioned via JavaScript on load/resize)
	sb.WriteString(`
  <!-- Notes Toggle (right-aligned via JavaScript) -->
  <rect x="364" y="91" width="130" height="33" rx="4"
        fill="#3498db" stroke="#2980b9" stroke-width="1"
        style="cursor:pointer" onclick="toggleNotes()"
        id="notes-toggle"/>
  <text x="377" y="113" font-family="Arial, sans-serif" font-size="14"
        fill="white" style="cursor:pointer; user-select: none"
        onclick="toggleNotes()" id="notes-text">
    Hide Notes
  </text>

  <!-- Fit to Width Toggle (right-aligned via JavaScript) -->
  <rect x="520" y="91" width="130" height="33" rx="4"
        fill="#3498db" stroke="#2980b9" stroke-width="1"
        style="cursor:pointer" onclick="toggleFitMode()"
        id="fit-toggle"/>
  <text x="533" y="113" font-family="Arial, sans-serif" font-size="14"
        fill="white" style="cursor:pointer; user-select: none"
        onclick="toggleFitMode()" id="fit-text">
    Native Size
  </text>
`)

	sb.WriteString(`

  <!-- Diagram Layers (positioned below header at y=140) -->
`)

	// Generate diagram layers
	for _, lvl := range levels {
		sb.WriteString(s.createDiagramLayer(lvl.name))
	}

	// Add JavaScript
	sb.WriteString(`
  <!-- Navigation Script -->
  <script type="text/ecmascript"><![CDATA[
    `)
	// Inject actual diagram dimensions
	sb.WriteString("const diagramData = {\n")
	diagramCount := 0
	for _, lvl := range levels {
		level := lvl.name
		if diagram, exists := s.diagrams[level]; exists {
			if diagramCount > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString(fmt.Sprintf("  '%s': { width: %.0f, height: %.0f, ratio: %.2f }",
				level, diagram.width, diagram.height, diagram.aspectRatio))
			diagramCount++
		}
	}
	sb.WriteString("\n};\n\n")

	// Inject available levels list
	sb.WriteString("const availableLevels = [")
	levelCount := 0
	for _, lvl := range levels {
		level := lvl.name
		if _, exists := s.diagrams[level]; exists {
			if levelCount > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("'%s'", level))
			levelCount++
		}
	}
	sb.WriteString("];\n\n")

	sb.Write(jsContent)
	sb.WriteString(`
  ]]></script>

</svg>`)

	return sb.String()
}

func (s *SVGStacker) createDiagramLayer(level string) string {
	diagram, exists := s.diagrams[level]
	if !exists {
		return ""
	}

	return fmt.Sprintf(`
  <!-- %s layer -->
  <g id="layer-%s" style="display:none">
    <rect x="5" y="145" width="99999" height="99999" fill="white" stroke="#ddd" stroke-width="1" rx="5" id="container-%s"/>
    <g id="diagram-%s">
      <svg viewBox="%s" x="10" y="150" width="99999" height="99999" preserveAspectRatio="xMidYMin meet">
        %s
      </svg>
    </g>
  </g>`, level, level, level, level, diagram.viewBox, diagram.content)
}
