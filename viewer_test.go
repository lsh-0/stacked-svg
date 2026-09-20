package main

import (
	"encoding/json"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// browsers tried in order; the first on PATH runs the viewer tests
var chromium_candidates = []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"}

// where the driver script leaves its observations for the test to read
var results_pattern = regexp.MustCompile(`(?s)<pre id="results">(.*?)</pre>`)

// runs `driver`, a JavaScript body that fills the object `r`, against the generated document
// for `case_dir` in headless Chromium and returns the observations. Skips when no browser
// is on PATH.
func run_viewer(t *testing.T, case_dir string, driver string) map[string]any {
	t.Helper()
	var browser string
	for _, candidate := range chromium_candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			browser = path
			break
		}
	}
	if browser == "" {
		t.Skipf("viewer test skipped: none of %v found on PATH", chromium_candidates)
	}

	diagrams, err := load(case_dir)
	if err != nil {
		t.Fatal(err)
	}
	document, err := stack("Viewer test", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), diagrams)
	if err != nil {
		t.Fatal(err)
	}
	// the document is inlined into an HTML page, so its XML declaration must go
	document = strings.TrimPrefix(document, `<?xml version="1.0" encoding="UTF-8"?>`)

	harness := "<html><body>" + document + `<pre id="results"></pre>
<script>
setTimeout(function () {
  var r = {};
  try {
` + driver + `
  } catch (e) {
    r.error = String(e && e.stack || e);
  }
  document.getElementById('results').textContent = JSON.stringify(r);
}, 50);
</script></body></html>`
	page := filepath.Join(t.TempDir(), "harness.html")
	if err := os.WriteFile(page, []byte(harness), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(browser,
		"--headless=new", "--no-sandbox", "--disable-gpu",
		"--virtual-time-budget=3000", "--dump-dom", "file://"+page)
	dump, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s failed: %v", browser, err)
	}
	match := results_pattern.FindSubmatch(dump)
	if match == nil {
		t.Fatalf("no results block in browser output:\n%s", dump)
	}
	var results map[string]any
	if err := json.Unmarshal([]byte(html.UnescapeString(string(match[1]))), &results); err != nil {
		t.Fatalf("results are not JSON: %v\n%s", err, match[1])
	}
	if script_error, ok := results["error"]; ok {
		t.Fatalf("driver script failed: %v", script_error)
	}
	return results
}

// JavaScript that lists the visible level names
const visible_levels_js = `
  var visible = function () {
    return availableLevels.filter(function (l) {
      var layer = document.getElementById('layer-' + l);
      return layer && layer.style.display !== 'none';
    });
  };`

func TestViewerInitialState(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), visible_levels_js+`
  r.visible = visible();
  r.buttons = {};
  availableLevels.forEach(function (l) {
    r.buttons[l] = document.getElementById('nav-' + l).getAttribute('fill');
  });`)

	expected_visible := `["context"]`
	if actual_visible, _ := json.Marshal(actual["visible"]); string(actual_visible) != expected_visible {
		t.Errorf("visible layers: got %s, want %s", actual_visible, expected_visible)
	}
	buttons := actual["buttons"].(map[string]any)
	for level, fill := range buttons {
		expected := "#3498db"
		if level == "context" {
			expected = "#e74c3c"
		}
		if fill != expected {
			t.Errorf("button %s fill: got %v, want %s", level, fill, expected)
		}
	}
}

// switching levels repeatedly must not add hitboxes or backgrounds again
func TestViewerEnhancesEachLayerOnce(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), `
  for (var i = 0; i < 10; i++) {
    showLevel(i % 2 === 0 ? 'container' : 'context');
  }
  r.layers = {};
  ['context', 'container'].forEach(function (l) {
    var layer = document.getElementById('layer-' + l);
    var labelled = Array.from(layer.querySelectorAll('g.link')).filter(function (g) {
      return g.querySelectorAll('text').length > 0;
    }).length;
    r.layers[l] = {
      labelledLinks: labelled,
      hitboxes: layer.querySelectorAll('.label-hitbox').length,
      backgrounds: layer.querySelectorAll('.text-bg').length
    };
  });`)

	for level, raw := range actual["layers"].(map[string]any) {
		counts := raw.(map[string]any)
		labelled := counts["labelledLinks"]
		if labelled == float64(0) {
			t.Errorf("%s: fixture has no labelled links, test is vacuous", level)
		}
		if counts["hitboxes"] != labelled || counts["backgrounds"] != labelled {
			t.Errorf("%s: %v labelled links but %v hitboxes and %v backgrounds", level, labelled, counts["hitboxes"], counts["backgrounds"])
		}
	}
}
