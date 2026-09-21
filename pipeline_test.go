package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writes the three required levels into a fresh directory and returns it
func svg_input_dir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	createTestSVGFiles(t, dir)
	return dir
}

func TestRunWritesToStdoutByDefault(t *testing.T) {
	given := svg_input_dir(t)
	var stdout strings.Builder

	if err := run(given, "", "Title", &stdout); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actual := stdout.String()
	if !strings.HasPrefix(actual, `<?xml version="1.0" encoding="UTF-8"?>`) || !strings.Contains(actual, "</svg>") {
		t.Errorf("stdout does not hold a document:\n%.200s", actual)
	}
	if entries, _ := os.ReadDir(given); len(entries) != 3 {
		t.Errorf("run wrote into the input directory: %v", entries)
	}
}

func TestRunWritesToFileAndNotStdout(t *testing.T) {
	given := svg_input_dir(t)
	output := filepath.Join(t.TempDir(), "out.svg")
	var stdout strings.Builder

	if err := run(given, output, "Title", &stdout); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout.Len() != 0 {
		t.Errorf("expected nothing on stdout, got %d bytes", stdout.Len())
	}
	actual, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	if !strings.Contains(string(actual), "</svg>") {
		t.Error("output file does not hold a document")
	}
}

func TestRenderWithoutPlantuml(t *testing.T) {
	t.Setenv("PATH", "")

	_, _, err := render("examples")

	if err == nil || !strings.Contains(err.Error(), "plantuml not found in PATH") {
		t.Fatalf("expected a plantuml-not-found error, got %v", err)
	}
}

// renders the example sources with the real PlantUML; skipped when it is not installed
func TestRenderExamples(t *testing.T) {
	if _, err := exec.LookPath("plantuml"); err != nil {
		t.Skip("plantuml not on PATH")
	}

	svg_dir, cleanup, err := render("examples")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	diagrams, err := load(svg_dir)
	if err != nil {
		t.Fatalf("load rendered output: %v", err)
	}
	cleanup()

	if len(diagrams) != 4 {
		t.Errorf("expected 4 diagrams from examples/, got %d", len(diagrams))
	}
	if _, err := os.Stat(svg_dir); !os.IsNotExist(err) {
		t.Errorf("cleanup left %s in place", svg_dir)
	}
	if stray, _ := filepath.Glob(filepath.Join("examples", "0?-*.svg")); len(stray) != 0 {
		t.Errorf("rendered files left in the input directory: %v", stray)
	}
}
