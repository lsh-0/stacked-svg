package main

import (
	"bytes"
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

type SVGStacker struct {
	diagrams   map[string]DiagramInfo
	svgWidth   int
	svgHeight  int
	inputDir   string
	outputFile string
	cssOnly    bool
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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: svg-stacker <directory> [-o output.svg] [--css-only]\n")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputFile := ""
	cssOnly := false

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-o":
			if i+1 < len(os.Args) {
				outputFile = os.Args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "Error: -o requires an argument\n")
				os.Exit(1)
			}
		case "--css-only":
			cssOnly = true
		}
	}

	stacker := NewSVGStacker(inputDir, outputFile, cssOnly)
	if err := stacker.CreateStackedSVG(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewSVGStacker(inputDir, outputFile string, cssOnly bool) *SVGStacker {
	return &SVGStacker{
		diagrams:   make(map[string]DiagramInfo),
		svgWidth:   800,
		svgHeight:  600,
		inputDir:   inputDir,
		outputFile: outputFile,
		cssOnly:    cssOnly,
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
	plantumlPath := "/home/user/bin/plantuml"
	args := []string{"-tsvg", "-o", tempDir}
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
	info.content = s.prettyPrintXML(cleanedContent)

	return info, nil
}

func (s *SVGStacker) prettyPrintXML(content string) string {
	// Wrap in a root element for parsing
	wrapped := "<root>" + content + "</root>"

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

	// Determine next level for navigation
	nextLevel := s.getNextLevel(currentLevel)
	_ = nextLevel // May be unused in CSS-only mode

	// For CSS-only mode, we need to ensure clean XML structure
	if s.cssOnly {
		// Use proper XML parsing to remove <a> tags
		content = s.removeATagsWithXML(content)
	} else {
		// JavaScript mode: add onclick handlers and clean up <a> tags
		aTagRegex := regexp.MustCompile(`(<g[^>]*>)\s*<a\s+[^>]*href="[^"]*"[^>]*>(.*?)</a>`)
		content = aTagRegex.ReplaceAllStringFunc(content, func(match string) string {
			submatches := aTagRegex.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				gTag := submatches[1]          // <g ...>
				contentInside := submatches[2] // content inside <a>

				// JavaScript mode: add onclick to the g element
				return strings.Replace(gTag, ">", ` onclick="navigateDown()" style="cursor:pointer;">`, 1) + contentInside
			}
			return match
		})

		// Clean up any remaining <a> tags
		content = regexp.MustCompile(`<a\s+[^>]*>`).ReplaceAllString(content, "")
		content = strings.ReplaceAll(content, "</a>", "")
	}

	return content
}

func (s *SVGStacker) removeATagsWithXML(content string) string {
	// Wrap content in a root element to make it valid XML
	wrappedContent := "<root>" + content + "</root>"

	// Parse the XML
	decoder := xml.NewDecoder(strings.NewReader(wrappedContent))
	var result strings.Builder

	// Track if we're inside an <a> tag
	var aTagDepth int

	for {
		token, err := decoder.Token()
		if err != nil {
			// If XML parsing fails, fall back to the original content without <a> tag removal
			// This means CSS-only mode won't have navigation but will have valid XML
			return content
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "a" {
				aTagDepth++
				// Skip the <a> start tag
				continue
			} else if aTagDepth == 0 {
				// Only output non-<a> start elements when not inside <a> tags
				result.WriteString("<")
				result.WriteString(t.Name.Local)
				for _, attr := range t.Attr {
					result.WriteString(" ")
					result.WriteString(attr.Name.Local)
					result.WriteString(`="`)
					// Properly escape attribute values
					escaped := strings.ReplaceAll(attr.Value, `"`, `&quot;`)
					escaped = strings.ReplaceAll(escaped, `&`, `&amp;`)
					escaped = strings.ReplaceAll(escaped, `<`, `&lt;`)
					escaped = strings.ReplaceAll(escaped, `>`, `&gt;`)
					result.WriteString(escaped)
					result.WriteString(`"`)
				}
				result.WriteString(">")
			}
		case xml.EndElement:
			if t.Name.Local == "a" {
				aTagDepth--
				// Skip the </a> end tag
				continue
			} else if aTagDepth == 0 {
				// Only output non-<a> end elements when not inside <a> tags
				result.WriteString("</")
				result.WriteString(t.Name.Local)
				result.WriteString(">")
			}
		case xml.CharData:
			if aTagDepth == 0 {
				// Only output character data when not inside <a> tags
				result.Write(t)
			}
		case xml.Comment:
			if aTagDepth == 0 {
				result.WriteString("<!--")
				result.Write(t)
				result.WriteString("-->")
			}
		}
	}

	// Remove the wrapper root element
	resultStr := result.String()
	if strings.HasPrefix(resultStr, "<root>") && strings.HasSuffix(resultStr, "</root>") {
		resultStr = resultStr[6 : len(resultStr)-7]
	}

	return resultStr
}

func (s *SVGStacker) getNextLevel(currentLevel string) string {
	switch currentLevel {
	case "context":
		return "container"
	case "container":
		return "component"
	case "component":
		return "code"
	default:
		return ""
	}
}

func (s *SVGStacker) buildStackedSVG() string {
	levels := []string{"context", "container", "component", "code"}

	var jsContent []byte
	if !s.cssOnly {
		// Load JavaScript for interactive mode
		var err error
		jsContent, err = os.ReadFile("navigation.js")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load navigation.js: %v\n", err)
			jsContent = []byte("// Navigation script not found")
		}
	}

	var sb strings.Builder

	// SVG Header
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" 
     xmlns:xlink="http://www.w3.org/1999/xlink"
     width="100%" 
     height="100%" 
     style="background: #f8f9fa; display: block; min-height: 100vh;">
     
  <title>Stacked C4 Architecture Diagrams</title>`)

	// Add CSS styles for CSS-only mode
	if s.cssOnly {
		sb.WriteString(`
  <style>
    /* CSS-only navigation using :target pseudo-class */
    .layer { 
      display: none; 
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
    }
    .layer:target { display: block; }
    
    /* Show context layer by default when no fragment */
    #layer-context { display: block; }
    
    /* Hide default layer when any layer is targeted */
    .layer:target ~ #layer-context:not(:target) { display: none; }
    
    /* Navigation button styles */
    .nav-button { 
      cursor: pointer; 
      transition: fill 0.2s; 
    }
    .nav-button:hover { 
      fill: #2980b9 !important; 
    }
  </style>`)
	}

	sb.WriteString(`

  <!-- Navigation Header -->
  <rect x="0" y="0" width="100%" height="80" fill="#2c3e50"/>
  <text x="26" y="33" font-family="Arial, sans-serif" font-size="21" font-weight="bold" fill="white">
    🏗️ Stacked C4 Architecture
  </text>
  <text x="26" y="59" font-family="Arial, sans-serif" font-size="16" fill="#ecf0f1" id="breadcrumb">
    Context Level
  </text>

  <!-- Navigation Buttons -->
`)

	// Generate navigation buttons (only for levels that exist)
	buttonIndex := 0
	for _, level := range levels {
		if _, exists := s.diagrams[level]; !exists {
			continue // Skip button if diagram doesn't exist
		}

		x := 26 + buttonIndex*117
		buttonIndex++

		if s.cssOnly {
			// CSS-only version using <a> tags with href fragments
			sb.WriteString(fmt.Sprintf(`  <a href="#layer-%s">
    <rect x="%d" y="91" width="104" height="33" rx="4"
          fill="#3498db" stroke="#2980b9" stroke-width="1"
          class="nav-button"/>
    <text x="%d" y="113" font-family="Arial, sans-serif" font-size="14"
          fill="white" style="cursor:pointer; user-select: none">
      %s
    </text>
  </a>
`, level, x, x+13, strings.Title(level)))
		} else {
			// JavaScript version
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
	}

	// Add fit-to-width toggle button (JavaScript mode only)
	// Positioned far right, will be adjusted by JavaScript on load/resize
	if !s.cssOnly {
		sb.WriteString(`
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

  <!-- Instructions -->
  <text x="390" y="113" font-family="Arial, sans-serif" font-size="13" fill="#7f8c8d" text-anchor="end" id="instructions">
    Click buttons or diagram elements to navigate
  </text>
`)
	}

	sb.WriteString(`

  <!-- Diagram Layers -->
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

	displayStyle := ""
	if !s.cssOnly {
		displayStyle = ` style="display:none"`
	}

	return fmt.Sprintf(`
  <!-- %s layer -->
  <g id="layer-%s" class="layer"%s>
    <rect x="5" y="140" width="calc(100%% - 10px)" height="calc(100vh - 160px)" fill="white" stroke="#ddd" stroke-width="1" rx="5" id="container-%s"/>
    <svg x="10" y="145" width="calc(100%% - 20px)" height="calc(100vh - 170px)" viewBox="%s" preserveAspectRatio="xMidYMid meet" id="diagram-%s">
      %s
    </svg>
  </g>`, level, level, displayStyle, level, diagram.viewBox, level, diagram.content)
}
