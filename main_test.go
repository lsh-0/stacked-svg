package main

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedSVGIsValidXMLStreaming(t *testing.T) {
	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Test both JavaScript and CSS-only modes
	modes := []struct {
		name    string
		cssOnly bool
		suffix  string
	}{
		{"JavaScript", false, ""},
		{"CSS-only", true, "-css"},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			outputFile := filepath.Join(tempDir, "test-stacked-c4"+mode.suffix+".svg")

			stacker := NewSVGStacker("examples/output", outputFile, mode.cssOnly)

			// Generate the SVG
			err := stacker.CreateStackedSVG()
			if err != nil {
				t.Fatalf("Failed to create stacked SVG: %v", err)
			}

			// Read the generated SVG
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read generated SVG: %v", err)
			}

			// Validate XML structure using streaming parser (more detailed error reporting)
			decoder := xml.NewDecoder(strings.NewReader(string(content)))
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("Generated SVG is not valid XML in %s mode: %v", mode.name, err)

					// Try to provide helpful context about where the error occurred
					lines := strings.Split(string(content), "\n")
					if xmlErr, ok := err.(*xml.SyntaxError); ok {
						lineNum := int(xmlErr.Line) - 1
						if lineNum >= 0 && lineNum < len(lines) {
							t.Errorf("Error at line %d: %s", xmlErr.Line, lines[lineNum])
							// Show some context around the error
							start := max(0, lineNum-2)
							end := min(len(lines), lineNum+3)
							for i := start; i < end; i++ {
								marker := "   "
								if i == lineNum {
									marker = ">>>"
								}
								t.Errorf("%s %d: %s", marker, i+1, lines[i])
							}
						}
					}
					return // Stop processing on first error
				}
			}
		})
	}
}

func TestActualGeneratedFiles(t *testing.T) {
	// Test the actual files generated in examples/output
	testCases := []struct {
		file string
		name string
	}{
		{"examples/output/stacked-c4.svg", "JavaScript"},
		{"examples/output/stacked-c4-css.svg", "CSS-only"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := os.ReadFile(tc.file)
			if err != nil {
				t.Skipf("File %s not found, skipping test", tc.file)
				return
			}

			// Validate XML structure using streaming parser
			decoder := xml.NewDecoder(strings.NewReader(string(content)))
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("File %s is not valid XML: %v", tc.file, err)

					// Try to provide helpful context about where the error occurred
					lines := strings.Split(string(content), "\n")
					if xmlErr, ok := err.(*xml.SyntaxError); ok {
						lineNum := int(xmlErr.Line) - 1
						if lineNum >= 0 && lineNum < len(lines) {
							t.Errorf("Error at line %d: %s", xmlErr.Line, lines[lineNum])
							// Show some context around the error
							start := max(0, lineNum-2)
							end := min(len(lines), lineNum+3)
							for i := start; i < end; i++ {
								marker := "   "
								if i == lineNum {
									marker = ">>>"
								}
								t.Errorf("%s %d: %s", marker, i+1, lines[i])
							}
						}
					}
					return // Stop processing on first error
				}
			}
		})
	}
}

func TestGeneratedSVGIsValidXML(t *testing.T) {
	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Test both JavaScript and CSS-only modes
	modes := []struct {
		name    string
		cssOnly bool
		suffix  string
	}{
		{"JavaScript", false, ""},
		{"CSS-only", true, "-css"},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			outputFile := filepath.Join(tempDir, "test-stacked-c4"+mode.suffix+".svg")

			stacker := NewSVGStacker("examples/output", outputFile, mode.cssOnly)

			// Generate the SVG
			err := stacker.CreateStackedSVG()
			if err != nil {
				t.Fatalf("Failed to create stacked SVG: %v", err)
			}

			// Read the generated SVG
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read generated SVG: %v", err)
			}

			// Validate XML structure using streaming parser (more detailed error reporting)
			decoder := xml.NewDecoder(strings.NewReader(string(content)))
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("Generated SVG is not valid XML in %s mode: %v", mode.name, err)

					// Try to provide helpful context about where the error occurred
					lines := strings.Split(string(content), "\n")
					if xmlErr, ok := err.(*xml.SyntaxError); ok {
						lineNum := int(xmlErr.Line) - 1
						if lineNum >= 0 && lineNum < len(lines) {
							t.Errorf("Error at line %d: %s", xmlErr.Line, lines[lineNum])
							// Show some context around the error
							start := max(0, lineNum-2)
							end := min(len(lines), lineNum+3)
							for i := start; i < end; i++ {
								marker := "   "
								if i == lineNum {
									marker = ">>>"
								}
								t.Errorf("%s %d: %s", marker, i+1, lines[i])
							}
						}
					}
					return // Stop processing on first error
				}
			}
		})
	}
}

func TestCleanDiagramContentPreservesXMLStructure(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		level       string
		cssOnly     bool
		expectValid bool
	}{
		{
			name:        "JavaScript mode with clickable entity",
			input:       `<g class="entity"><a href="original.svg">content</a></g>`,
			level:       "context",
			cssOnly:     false,
			expectValid: true,
		},
		{
			name:        "CSS-only mode with clickable entity",
			input:       `<g class="entity"><a href="original.svg">content</a></g>`,
			level:       "context",
			cssOnly:     true,
			expectValid: true,
		},
		{
			name:        "Complex nested structure",
			input:       `<g class="entity"><a href="test.svg"><rect/><text>Test</text></a><rect/></g>`,
			level:       "container",
			cssOnly:     true,
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stacker := &SVGStacker{cssOnly: tc.cssOnly}
			result := stacker.cleanDiagramContent(tc.input, tc.level)

			// Wrap in a minimal SVG structure for XML validation
			xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg">` + result + `</svg>`

			var xmlDoc interface{}
			err := xml.Unmarshal([]byte(xmlContent), &xmlDoc)

			if tc.expectValid && err != nil {
				t.Errorf("Expected valid XML but got error: %v", err)
				t.Errorf("Generated content: %s", result)
			} else if !tc.expectValid && err == nil {
				t.Errorf("Expected invalid XML but validation passed")
			}
		})
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
