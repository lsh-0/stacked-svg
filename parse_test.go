package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDiagramDimensions(t *testing.T) {
	tests := []struct {
		name             string
		given            string
		expected_viewbox string
		expected_width   float64
		expected_height  float64
	}{
		{
			name:             "plain numbers",
			given:            `<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600" viewBox="0 0 800 600"/>`,
			expected_viewbox: "0 0 800 600", expected_width: 800, expected_height: 600,
		},
		{
			name:             "px suffix",
			given:            `<svg xmlns="http://www.w3.org/2000/svg" width="494px" height="562px" viewBox="0 0 494 562"/>`,
			expected_viewbox: "0 0 494 562", expected_width: 494, expected_height: 562,
		},
		{
			name:             "absent dimensions use defaults",
			given:            `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`,
			expected_viewbox: "0 0 400 300", expected_width: 400, expected_height: 300,
		},
		{
			name:             "leading declaration and comment are skipped",
			given:            `<?xml version="1.0"?><!-- made by hand --><svg width="10" height="20" viewBox="0 0 10 20"/>`,
			expected_viewbox: "0 0 10 20", expected_width: 10, expected_height: 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := parse_diagram(levels[0], []byte(tt.given))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual.viewBox != tt.expected_viewbox || actual.width != tt.expected_width || actual.height != tt.expected_height {
				t.Errorf("got viewBox=%q width=%v height=%v", actual.viewBox, actual.width, actual.height)
			}
		})
	}
}

func TestParseDiagramRejectsBadInput(t *testing.T) {
	tests := []struct {
		name         string
		given        string
		expected_err string
	}{
		{"unclosed element", `<svg><g></svg>`, "element <g> closed by </svg>"},
		{"wrong root", `<html><body/></html>`, "root element is <html>, not <svg>"},
		{"no element", `<?xml version="1.0"?>`, "no root <svg> element"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse_diagram(levels[0], []byte(tt.given))
			if err == nil || !strings.Contains(err.Error(), tt.expected_err) {
				t.Errorf("expected error containing %q, got %v", tt.expected_err, err)
			}
		})
	}
}

// the error for a malformed input file names the file
func TestMalformedInputFileIsNamed(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"01-context.svg":   `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`,
		"02-container.svg": `<svg xmlns="http://www.w3.org/2000/svg"><g></svg>`,
		"03-component.svg": `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(dir, "out.svg")
	err := run(dir, output, "")
	if err == nil || !strings.Contains(err.Error(), "02-container.svg") {
		t.Fatalf("expected error naming 02-container.svg, got %v", err)
	}
	if _, stat_err := os.Stat(output); !os.IsNotExist(stat_err) {
		t.Error("output file was written despite the error")
	}
}

func body_of(t *testing.T, given string) string {
	t.Helper()
	actual, err := parse_diagram(levels[0], []byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`+given+`</svg>`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := decode_tokens([]byte("<r xmlns:xlink=\"x\">" + actual.body + "</r>")); err != nil {
		t.Fatalf("body is not well-formed: %v\n%s", err, actual.body)
	}
	return actual.body
}

func TestParseDiagramRewritesContent(t *testing.T) {
	tests := []struct {
		name     string
		given    string
		contains []string
		omits    []string
	}{
		{
			name:     "script and title removed",
			given:    `<title>T</title><script type="text/javascript">alert(1)</script><g><rect/></g>`,
			contains: []string{"<rect"},
			omits:    []string{"<title", "<script", "alert"},
		},
		{
			name:     "processing instruction removed",
			given:    `<?plantuml 1.2026.8?><g><rect/></g>`,
			contains: []string{"<rect"},
			omits:    []string{"<?plantuml"},
		},
		{
			name:     "linked group becomes clickable",
			given:    `<g class="entity" id="ent0002"><a target="_top" href="02-container.svg" xlink:href="02-container.svg"><rect/><text>System</text></a></g>`,
			contains: []string{`<g class="entity" id="ent0002" onclick="navigateDown()" style="cursor:pointer;">`, "<rect", "System"},
			omits:    []string{"<a ", "</a>"},
		},
		{
			name:     "existing style is extended",
			given:    `<g style="opacity:1;"><a href="x.svg"><rect/></a></g>`,
			contains: []string{`style="opacity:1; cursor:pointer;"`},
		},
		{
			name:     "other anchors are unwrapped",
			given:    `<g class="entity"><rect/><a href="https://example.invalid"><text>Docs</text></a></g>`,
			contains: []string{"<rect", "<text>Docs</text>"},
			omits:    []string{"<a ", "onclick"},
		},
		{
			name:     "embedded image keeps xlink:href",
			given:    `<g><image width="52" height="52" xlink:href="data:image/png;base64,AAAA"/></g>`,
			contains: []string{`xlink:href="data:image/png;base64,AAAA"`, `xmlns:xlink="http://www.w3.org/1999/xlink"`},
		},
		{
			name:     "comments and data attributes preserved",
			given:    `<!--entity customer--><g class="entity" data-qualified-name="customer" id="ent0001"><rect/></g>`,
			contains: []string{"<!--entity customer-->", `data-qualified-name="customer"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := body_of(t, tt.given)
			for _, want := range tt.contains {
				if !strings.Contains(actual, want) {
					t.Errorf("missing %q in:\n%s", want, actual)
				}
			}
			for _, unwanted := range tt.omits {
				if strings.Contains(actual, unwanted) {
					t.Errorf("unexpected %q in:\n%s", unwanted, actual)
				}
			}
		})
	}
}

func TestParseDiagramTagsNotes(t *testing.T) {
	tests := []struct {
		name     string
		given    string
		contains []string
		omits    []string
	}{
		{
			name: "current PlantUML ids via data-entity attributes",
			given: `<g class="entity" data-qualified-name="GMN8" id="ent0009"><path d="M0,0" fill="#FEFFDD"/><text>note</text></g>` +
				`<g class="link" data-entity-1="ent0009" data-entity-2="ent0002" id="lnk8"><path d="M1,1"/></g>` +
				`<g class="link" data-entity-1="ent0001" data-entity-2="ent0002" id="lnk5"><path d="M2,2"/></g>`,
			contains: []string{`class="entity note"`, `<g class="link note-link" data-entity-1="ent0009"`},
			omits:    []string{`<g class="link note-link" data-entity-1="ent0001"`},
		},
		{
			name: "older PlantUML ids via link id substring",
			given: `<g class="entity" id="entity_GMN49"><path d="M0,0" fill="#FEFFDD"/></g>` +
				`<g class="link" id="link_GMN49_database"><path d="M1,1"/></g>` +
				`<g class="link" id="link_web_app_database"><path d="M2,2"/></g>`,
			contains: []string{`class="entity note"`, `<g class="link note-link" id="link_GMN49_database"`},
			omits:    []string{`note-link" id="link_web_app_database"`},
		},
		{
			name:     "entity without the note fill is not a note",
			given:    `<g class="entity" id="ent0001"><rect fill="#08427B"/></g>`,
			omits:    []string{"note"},
			contains: []string{`class="entity"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := body_of(t, tt.given)
			for _, want := range tt.contains {
				if !strings.Contains(actual, want) {
					t.Errorf("missing %q in:\n%s", want, actual)
				}
			}
			for _, unwanted := range tt.omits {
				if strings.Contains(actual, unwanted) {
					t.Errorf("unexpected %q in:\n%s", unwanted, actual)
				}
			}
		})
	}
}
