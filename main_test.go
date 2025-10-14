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
	outputFile := filepath.Join(tempDir, "test-stacked-c4.svg")

	stacker := NewSVGStacker("examples/output", outputFile, "Test Diagram")

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
			t.Errorf("Generated SVG is not valid XML: %v", err)

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
}

func TestActualGeneratedFiles(t *testing.T) {
	// Test that we can generate a valid SVG from examples directory
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "actual-test.svg")

	stacker := NewSVGStacker("examples/output", outputFile, "Test")
	err := stacker.CreateStackedSVG()
	if err != nil {
		t.Skipf("Could not generate SVG (no example files?): %v", err)
		return
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Validate XML structure using streaming parser
	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Generated SVG is not valid XML: %v", err)

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
}

func TestGeneratedSVGContainsExpectedElements(t *testing.T) {
	// Create a temporary directory for test output
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "test-stacked-c4.svg")

	stacker := NewSVGStacker("examples/output", outputFile, "Test Architecture")

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

	contentStr := string(content)

	// Check for essential elements
	expectedElements := []string{
		"<svg",
		"Stacked C4 Architecture Diagrams",
		"diagram-",
		"layer-",
		"showLevel",
		"resizeContainers",
		"Navigation Header",
		"JavaScript", // Navigation script section
	}

	for _, element := range expectedElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Generated SVG missing expected element: %s", element)
		}
	}

	// Validate it's valid XML
	decoder := xml.NewDecoder(strings.NewReader(contentStr))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Generated SVG is not valid XML: %v", err)
			return
		}
	}
}

func TestCleanDiagramContentPreservesXMLStructure(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		level       string
		expectValid bool
	}{
		{
			name:        "Clickable entity with link",
			input:       `<g class="entity"><a href="original.svg">content</a></g>`,
			level:       "context",
			expectValid: true,
		},
		{
			name:        "Complex nested structure",
			input:       `<g class="entity"><a href="test.svg"><rect/><text>Test</text></a><rect/></g>`,
			level:       "container",
			expectValid: true,
		},
		{
			name:        "Script tag removal",
			input:       `<g><script>alert("test")</script><rect/></g>`,
			level:       "component",
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stacker := &SVGStacker{}
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

			// Verify script tags were removed
			if strings.Contains(result, "<script") {
				t.Errorf("Script tags should be removed from output")
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
