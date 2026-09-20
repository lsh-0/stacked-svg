package main

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createTestSVGFiles creates minimal valid SVG files for testing.
func createTestSVGFiles(t *testing.T, dir string) {
	t.Helper()

	testSVGs := map[string]string{
		"01-context.svg": `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="400" height="300" viewBox="0 0 400 300">
  <title>Test Context</title>
  <rect x="10" y="10" width="380" height="280" fill="white" stroke="black"/>
  <text x="200" y="150" text-anchor="middle">Context Diagram</text>
  <g class="link">
    <path d="M 100,100 L 300,200" stroke="#666" stroke-width="1"/>
    <text x="200" y="150" fill="#666">Test Link</text>
  </g>
</svg>`,
		"02-container.svg": `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="500" height="400" viewBox="0 0 500 400">
  <title>Test Container</title>
  <rect x="10" y="10" width="480" height="380" fill="white" stroke="black"/>
  <text x="250" y="200" text-anchor="middle">Container Diagram</text>
</svg>`,
		"03-component.svg": `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="300" height="200" viewBox="0 0 300 200">
  <text x="150" y="100" text-anchor="middle">Component Diagram</text>
</svg>`,
	}

	for filename, content := range testSVGs {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test SVG %s: %v", filename, err)
		}
	}
}

func TestGeneratedSVGContainsExpectedElements(t *testing.T) {
	// Create a temporary directory for test input and output
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	// Create test SVG files
	createTestSVGFiles(t, inputDir)

	diagrams, err := load(inputDir)
	if err != nil {
		t.Fatalf("Failed to load diagrams: %v", err)
	}
	contentStr, err := stack("Test Architecture", time.Now(), diagrams)
	if err != nil {
		t.Fatalf("Failed to create stacked SVG: %v", err)
	}

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

// TestParseArgsSlice tests the parseArgsSlice function for argument parsing logic
func TestParseArgsSlice(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectErr    bool
		expectDir    string
		expectOutput string
		expectTitle  string
	}{
		{
			name:      "no arguments",
			args:      []string{},
			expectErr: true,
		},
		{
			name:      "help flag short",
			args:      []string{"-h"},
			expectErr: true,
		},
		{
			name:      "help flag long",
			args:      []string{"--help"},
			expectErr: true,
		},
		{
			name:      "version flag short",
			args:      []string{"-v"},
			expectErr: true,
		},
		{
			name:      "version flag long",
			args:      []string{"--version"},
			expectErr: true,
		},
		{
			name:      "directory only",
			args:      []string{"./examples"},
			expectErr: false,
			expectDir: "./examples",
		},
		{
			name:         "directory with output",
			args:         []string{"./examples", "--output", "out.svg"},
			expectErr:    false,
			expectDir:    "./examples",
			expectOutput: "out.svg",
		},
		{
			name:        "directory with title",
			args:        []string{"./examples", "--title", "My Title"},
			expectErr:   false,
			expectDir:   "./examples",
			expectTitle: "My Title",
		},
		{
			name:         "all options",
			args:         []string{"./examples", "--output", "out.svg", "--title", "My Title"},
			expectErr:    false,
			expectDir:    "./examples",
			expectOutput: "out.svg",
			expectTitle:  "My Title",
		},
		{
			name:      "output without value",
			args:      []string{"./examples", "--output"},
			expectErr: true,
		},
		{
			name:      "title without value",
			args:      []string{"./examples", "--title"},
			expectErr: true,
		},
		{
			name:      "unknown flag",
			args:      []string{"./examples", "--unknown"},
			expectErr: true,
		},
		{
			name:      "help in middle of args",
			args:      []string{"./examples", "-h", "--output", "out.svg"},
			expectErr: true,
		},
		{
			name:      "version in middle of args",
			args:      []string{"./examples", "--version"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputDir, outputFile, title, err := parseArgsSlice(tt.args)

			if (err != nil) != tt.expectErr {
				if tt.expectErr {
					t.Errorf("expected error, got nil")
				} else {
					t.Errorf("expected no error, got %v", err)
				}
			}

			if !tt.expectErr {
				if inputDir != tt.expectDir {
					t.Errorf("InputDir: got %q, want %q", inputDir, tt.expectDir)
				}

				if outputFile != tt.expectOutput {
					t.Errorf("OutputFile: got %q, want %q", outputFile, tt.expectOutput)
				}

				if title != tt.expectTitle {
					t.Errorf("Title: got %q, want %q", title, tt.expectTitle)
				}
			}
		})
	}
}

func TestDiscoverLevels(t *testing.T) {
	tests := []struct {
		name         string
		given        []string
		expected     []string // level names in order
		expected_err string
	}{
		{
			name:     "four numbered files",
			given:    []string{"01-context.svg", "02-container.svg", "03-component.svg", "04-code.svg"},
			expected: []string{"context", "container", "component", "code"},
		},
		{
			name:     "optional code level absent",
			given:    []string{"01-context.svg", "02-container.svg", "03-component.svg"},
			expected: []string{"context", "container", "component"},
		},
		{
			name:         "required container level absent",
			given:        []string{"01-context.svg", "03-component.svg"},
			expected_err: "missing required container level",
		},
		{
			name:         "substring names are ignored",
			given:        []string{"Context-Diagram.svg", "02-container.svg", "03-component.svg"},
			expected_err: "missing required context level",
		},
		{
			name:     "other extensions are ignored",
			given:    []string{"01-context.svg", "02-container.svg", "03-component.svg", "04-code.puml"},
			expected: []string{"context", "container", "component"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tt.given {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("<svg/>"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			found, err := discover_levels(dir, ".svg")

			if tt.expected_err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expected_err) {
					t.Fatalf("expected error containing %q, got %v", tt.expected_err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var actual []string
			for _, f := range found {
				actual = append(actual, f.level.name)
				if filepath.Dir(f.path) != dir {
					t.Errorf("path %q is not inside %q", f.path, dir)
				}
			}
			if strings.Join(actual, ",") != strings.Join(tt.expected, ",") {
				t.Errorf("got %v, want %v", actual, tt.expected)
			}
		})
	}
}

// TestTitleCase tests the titleCase function
func TestTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"context", "Context"},
		{"container", "Container"},
		{"component", "Component"},
		{"code", "Code"},
		{"hello world", "Hello world"},
		{"already Title", "Already Title"},
		{"", ""},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := titleCase(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
