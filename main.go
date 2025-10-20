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
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed navigation.js
var navigationJS string

type SVGStacker struct {
	diagrams   map[string]DiagramInfo
	svgWidth   int
	svgHeight  int
	inputDir   string
	outputFile string
	title      string
	tempDir    string
}

type DiagramInfo struct {
	content     string
	viewBox     string
	width       float64
	height      float64
	aspectRatio float64
}

func validateXML(content string) error {
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

var version = "unreleased"

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: svg-stacker [OPTIONS] <directory>

Combines C4 architecture diagrams (SVG or PlantUML) into a single interactive stacked SVG.

ARGUMENTS:
  <directory>         Directory containing SVG files or .puml files

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

func parseArgs() (inputDir, outputFile, title string, shouldExit bool, exitCode int) {
	if len(os.Args) < 2 {
		printUsage()
		return "", "", "", true, 1
	}

	// Check for help/version flags first
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage()
			return "", "", "", true, 0
		}
		if arg == "-v" || arg == "--version" {
			printVersion()
			return "", "", "", true, 0
		}
	}

	inputDir = os.Args[1]
	outputFile = ""
	title = ""

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--output":
			if i+1 < len(os.Args) {
				outputFile = os.Args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "Error: --output requires an argument\n")
				return "", "", "", true, 1
			}
		case "--title":
			if i+1 < len(os.Args) {
				title = os.Args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "Error: --title requires an argument\n")
				return "", "", "", true, 1
			}
		case "-h", "--help", "-v", "--version":
			// Already handled above
		default:
			// Unknown flag
			fmt.Fprintf(os.Stderr, "Error: unknown flag '%s'\n", os.Args[i])
			fmt.Fprintf(os.Stderr, "Use 'svg-stacker --help' for usage information\n")
			return "", "", "", true, 1
		}
	}

	return inputDir, outputFile, title, false, 0
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
		svgWidth:   800,
		svgHeight:  600,
		inputDir:   inputDir,
		outputFile: outputFile,
		title:      title,
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
	// Find all .puml files numbered 01-04
	pumlFiles, err := s.findNumberedPumlFiles()
	if err != nil {
		return err
	}

	if len(pumlFiles) < 3 {
		return fmt.Errorf("expected at least 3 numbered .puml files (01-*.puml through 03-*.puml), found %d", len(pumlFiles))
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

func (s *SVGStacker) findNumberedPumlFiles() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(s.inputDir, "*.puml"))
	if err != nil {
		return nil, err
	}

	// Filter and sort by number prefix (01-04, with 04 being optional)
	var numbered []string
	numberRegex := regexp.MustCompile(`^0[1-4]-.*\.puml$`)

	for _, file := range files {
		base := filepath.Base(file)
		if numberRegex.MatchString(base) {
			numbered = append(numbered, file)
		}
	}

	sort.Strings(numbered)
	return numbered, nil
}

func (s *SVGStacker) loadDiagrams() error {
	// Find all SVG files in the input directory
	files, err := filepath.Glob(filepath.Join(s.inputDir, "*.svg"))
	if err != nil {
		return err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		// Validate XML before processing
		if err := validateXML(string(content)); err != nil {
			return err
		}

		level := s.extractLevel(filepath.Base(file))
		if level == "unknown" {
			continue // Skip files that don't match C4 patterns
		}

		info, err := s.parseSVG(string(content), level)
		if err != nil {
			return err
		}

		s.diagrams[level] = info
	}

	if len(s.diagrams) == 0 {
		return fmt.Errorf("no C4 SVG files found")
	}

	return nil
}

func (s *SVGStacker) extractLevel(filename string) string {
	lower := strings.ToLower(filename)
	if strings.Contains(lower, "context") {
		return "context"
	}
	if strings.Contains(lower, "container") {
		return "container"
	}
	if strings.Contains(lower, "component") {
		return "component"
	}
	if strings.Contains(lower, "code") {
		return "code"
	}
	return "unknown"
}

func (s *SVGStacker) parseSVG(content string, level string) (DiagramInfo, error) {
	var info DiagramInfo

	// Extract SVG element attributes
	svgRegex := regexp.MustCompile(`<svg[^>]*>`)
	match := svgRegex.FindString(content)
	if match == "" {
		return info, fmt.Errorf("no SVG element")
	}

	// Extract viewBox
	viewBoxRegex := regexp.MustCompile(`viewBox="([^"]*)"`)
	if viewBoxMatch := viewBoxRegex.FindStringSubmatch(match); len(viewBoxMatch) > 1 {
		info.viewBox = viewBoxMatch[1]
	} else {
		info.viewBox = "0 0 400 300"
	}

	// Extract width and height
	widthRegex := regexp.MustCompile(`width="([^"]*)"`)
	heightRegex := regexp.MustCompile(`height="([^"]*)"`)

	var err error
	if widthMatch := widthRegex.FindStringSubmatch(match); len(widthMatch) > 1 {
		widthStr := strings.TrimSuffix(widthMatch[1], "px")
		info.width, err = strconv.ParseFloat(widthStr, 64)
		if err != nil {
			info.width = 400
		}
	} else {
		info.width = 400
	}

	if heightMatch := heightRegex.FindStringSubmatch(match); len(heightMatch) > 1 {
		heightStr := strings.TrimSuffix(heightMatch[1], "px")
		info.height, err = strconv.ParseFloat(heightStr, 64)
		if err != nil {
			info.height = 300
		}
	} else {
		info.height = 300
	}

	info.aspectRatio = info.width / info.height

	// Extract content between <svg> and </svg> more robustly
	// Find the end of the opening <svg> tag
	svgStartPos := strings.Index(content, "<svg")
	if svgStartPos == -1 {
		return info, fmt.Errorf("no <svg> tag")
	}

	svgTagEndPos := strings.Index(content[svgStartPos:], ">")
	if svgTagEndPos == -1 {
		return info, fmt.Errorf("malformed <svg> tag")
	}

	startIdx := svgStartPos + svgTagEndPos + 1
	endIdx := strings.LastIndex(content, "</svg>")

	if endIdx == -1 || endIdx <= startIdx {
		return info, fmt.Errorf("no </svg> tag")
	}

	rawContent := content[startIdx:endIdx]
	cleanedContent := s.cleanDiagramContent(rawContent, level)
	// Pretty-print the content for better readability (namespace context is preserved)
	info.content = s.prettyPrintXML(cleanedContent)

	return info, nil
}

func (s *SVGStacker) prettyPrintXML(content string) string {
	// Wrap in a root element with namespace declarations for parsing.
	// IMPORTANT: We must include xmlns:xlink here because the content we're formatting
	// is extracted from inside an <svg> tag (which normally declares xmlns:xlink).
	// Without this declaration, Go's xml.Encoder will mangle xlink:href attributes
	// by changing xmlns:xlink="http://www.w3.org/1999/xlink" to xmlns:xlink="xlink",
	// which breaks <image> elements that use xlink:href for embedded data URIs.
	wrapped := `<root xmlns:xlink="http://www.w3.org/1999/xlink">` + content + "</root>"

	var buf bytes.Buffer
	decoder := xml.NewDecoder(strings.NewReader(wrapped))
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("      ", "  ")

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// If parsing fails, return original content
			return content
		}

		// Skip the root wrapper element
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "root" {
			continue
		}
		if end, ok := token.(xml.EndElement); ok && end.Name.Local == "root" {
			continue
		}

		if err := encoder.EncodeToken(token); err != nil {
			return content
		}
	}

	if err := encoder.Flush(); err != nil {
		return content
	}

	return strings.TrimSpace(buf.String())
}

func (s *SVGStacker) cleanDiagramContent(content string, currentLevel string) string {
	// Remove scripts
	scriptRegex := regexp.MustCompile(`<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")

	// Add onclick handlers and clean up <a> tags
	aTagRegex := regexp.MustCompile(`(<g[^>]*>)\s*<a\s+[^>]*href="[^"]*"[^>]*>(.*?)</a>`)
	content = aTagRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := aTagRegex.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			gTag := submatches[1]          // <g ...>
			contentInside := submatches[2] // content inside <a>

			// Add onclick to the g element
			return strings.Replace(gTag, ">", ` onclick="navigateDown()" style="cursor:pointer;">`, 1) + contentInside
		}
		return match
	})

	// Clean up any remaining <a> tags
	content = regexp.MustCompile(`<a\s+[^>]*>`).ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, "</a>", "")

	return content
}

func (s *SVGStacker) buildStackedSVG() string {
	levels := []string{"context", "container", "component", "code"}

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
	for _, level := range levels {
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
`, x, level, level, x+13, level, strings.Title(level)))
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
	for _, level := range levels {
		sb.WriteString(s.createDiagramLayer(level))
	}

	// Add JavaScript
	sb.WriteString(`
  <!-- Navigation Script -->
  <script type="text/ecmascript"><![CDATA[
    `)
	// Inject actual diagram dimensions
	sb.WriteString("const diagramData = {\n")
	diagramCount := 0
	for _, level := range levels {
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
	for _, level := range levels {
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
		return fmt.Sprintf(`
  <!-- %s layer (not found) -->
  <g id="layer-%s" style="display:none">
    <rect x="50" y="120" width="700" height="450" fill="#ecf0f1" stroke="#bdc3c7"/>
    <text x="400" y="350" text-anchor="middle" font-family="Arial" font-size="16" fill="#7f8c8d">
      %s diagram not found
    </text>
  </g>`, level, level, strings.Title(level))
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
