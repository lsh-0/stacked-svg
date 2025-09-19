package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type SVGStacker struct {
	diagrams map[string]DiagramInfo
	svgWidth int
	svgHeight int
}

type DiagramInfo struct {
	content     string
	viewBox     string
	width       float64
	height      float64
	aspectRatio float64
}

func main() {
	stacker := NewSVGStacker()
	if err := stacker.CreateStackedSVG(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stacked SVG: %v\n", err)
		os.Exit(1)
	}
}

func NewSVGStacker() *SVGStacker {
	return &SVGStacker{
		diagrams:  make(map[string]DiagramInfo),
		svgWidth:  800,
		svgHeight: 600,
	}
}

func (s *SVGStacker) CreateStackedSVG() error {
	fmt.Println("📚 Creating self-contained stacked SVG...")
	
	// Load all SVG files
	if err := s.loadDiagrams(); err != nil {
		return fmt.Errorf("failed to load diagrams: %w", err)
	}
	
	// Create the master SVG
	stackedSVG := s.buildStackedSVG()
	
	// Write the result
	if err := os.WriteFile("output/stacked-c4.svg", []byte(stackedSVG), 0644); err != nil {
		return fmt.Errorf("failed to write stacked SVG: %w", err)
	}
	
	fmt.Println("✅ Created output/stacked-c4.svg")
	fmt.Println("📖 Open this file directly in any SVG viewer or browser")
	return nil
}

func (s *SVGStacker) loadDiagrams() error {
	svgFiles := []string{
		"output/01-context.svg",
		"output/02-container.svg", 
		"output/03-component.svg",
		"output/04-code.svg",
	}

	for _, file := range svgFiles {
		if _, err := os.Stat(file); err != nil {
			continue // Skip missing files
		}
		
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		
		level := s.extractLevel(file)
		info, err := s.parseSVG(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", file, err)
		}
		
		s.diagrams[level] = info
		fmt.Printf("  📄 Loaded %s level (%.0fx%.0f, ratio: %.2f)\n", 
			level, info.width, info.height, info.aspectRatio)
	}
	
	return nil
}

func (s *SVGStacker) extractLevel(filename string) string {
	if strings.Contains(filename, "01-context") {
		return "context"
	}
	if strings.Contains(filename, "02-container") {
		return "container"
	}
	if strings.Contains(filename, "03-component") {
		return "component"
	}
	if strings.Contains(filename, "04-code") {
		return "code"
	}
	return "unknown"
}

func (s *SVGStacker) parseSVG(content string) (DiagramInfo, error) {
	var info DiagramInfo
	
	// Extract SVG element attributes
	svgRegex := regexp.MustCompile(`<svg[^>]*>`)
	match := svgRegex.FindString(content)
	if match == "" {
		return info, fmt.Errorf("no SVG element found")
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
	
	// Extract content between <svg> and </svg>
	startIdx := strings.Index(content, ">") + 1
	endIdx := strings.LastIndex(content, "</svg>")
	if startIdx > 0 && endIdx > startIdx {
		info.content = content[startIdx:endIdx]
	} else {
		return info, fmt.Errorf("failed to extract SVG content")
	}
	
	// Clean the content
	info.content = s.cleanDiagramContent(info.content)
	
	return info, nil
}

func (s *SVGStacker) cleanDiagramContent(content string) string {
	// Remove scripts
	scriptRegex := regexp.MustCompile(`<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")
	
	// Find <a> tags and their parent elements to add click handlers
	// Look for pattern: <g ...><a href="...">content</a></g>
	aTagRegex := regexp.MustCompile(`(<g[^>]*)(>\s*<a\s+[^>]*href="[^"]*"[^>]*>)(.*?)</a>`)
	content = aTagRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := aTagRegex.FindStringSubmatch(match)
		if len(submatches) >= 4 {
			gTag := submatches[1]           // <g attributes
			contentInside := submatches[3]  // content inside <a>
			
			// Add onclick to the g element and remove the <a> tag
			return gTag + ` onclick="navigateDown()" style="cursor:pointer;">` + contentInside
		}
		return match
	})
	
	// Clean up any remaining <a> tags that might not have been caught
	content = regexp.MustCompile(`<a\s+[^>]*>`).ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, "</a>", "")
	
	return content
}

func (s *SVGStacker) buildStackedSVG() string {
	levels := []string{"context", "container", "component", "code"}
	
	// Load JavaScript
	jsContent, err := os.ReadFile("navigation.js")
	if err != nil {
		fmt.Printf("Warning: could not load navigation.js: %v\n", err)
		jsContent = []byte("// Navigation script not found")
	}
	
	var sb strings.Builder
	
	// SVG Header
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" 
     xmlns:xlink="http://www.w3.org/1999/xlink"
     width="100%" 
     height="100%" 
     viewBox="0 0 `)
	sb.WriteString(fmt.Sprintf("%d %d", s.svgWidth, s.svgHeight))
	sb.WriteString(`"
     preserveAspectRatio="xMidYMid meet"
     style="background: #f8f9fa; display: block; min-height: 100vh;">
     
  <title>Stacked C4 Architecture Diagrams</title>
  
  <!-- Navigation Header -->
  <rect x="0" y="0" width="`)
	sb.WriteString(strconv.Itoa(s.svgWidth))
	sb.WriteString(`" height="60" fill="#2c3e50"/>
  <text x="20" y="25" font-family="Arial, sans-serif" font-size="16" font-weight="bold" fill="white">
    🏗️ Stacked C4 Architecture
  </text>
  <text x="20" y="45" font-family="Arial, sans-serif" font-size="12" fill="#ecf0f1" id="breadcrumb">
    Context Level
  </text>
  
  <!-- Navigation Buttons -->
`)

	// Generate navigation buttons
	for i, level := range levels {
		x := 20 + i*90
		sb.WriteString(fmt.Sprintf(`  <rect x="%d" y="70" width="80" height="25" rx="3" 
        fill="#3498db" stroke="#2980b9" stroke-width="1" 
        style="cursor:pointer" onclick="showLevel('%s')" 
        id="nav-%s"/>
  <text x="%d" y="86" font-family="Arial, sans-serif" font-size="11" 
        fill="white" style="cursor:pointer; user-select: none" 
        onclick="showLevel('%s')">
    %s
  </text>
`, x, level, level, x+10, level, strings.Title(level)))
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
	sb.Write(jsContent)
	sb.WriteString(`
  ]]></script>
  
  <!-- Instructions -->
  <text x="450" y="86" font-family="Arial, sans-serif" font-size="10" fill="#7f8c8d">
    Click buttons or diagram elements to navigate • Self-contained SVG
  </text>
  
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
    <rect x="20" y="110" width="760" height="470" fill="white" stroke="#ddd" stroke-width="1" rx="5" id="container-%s"/>
    <svg x="30" y="120" width="740" height="450" viewBox="%s" preserveAspectRatio="xMidYMid meet" id="diagram-%s">
      %s
    </svg>
  </g>`, level, level, level, diagram.viewBox, level, diagram.content)
}