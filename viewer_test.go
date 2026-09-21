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

// JavaScript helpers shared by the interaction tests: element lookup and event dispatch
const interaction_js = `
  var hitboxes = function (level) {
    return Array.from(document.querySelectorAll('#layer-' + level + ' .label-hitbox'));
  };
  var pinned = function () {
    return Array.from(document.querySelectorAll('g.link.highlighted')).map(function (g) { return g.id; }).sort();
  };
  var mouse = function (el, type, opts) {
    el.dispatchEvent(new MouseEvent(type, Object.assign({ bubbles: true, cancelable: true }, opts || {})));
  };
  var key = function (name) {
    document.dispatchEvent(new KeyboardEvent('keydown', { key: name }));
  };`

func as_json(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func expect_json(t *testing.T, results map[string]any, key string, expected string) {
	t.Helper()
	if actual := as_json(t, results[key]); actual != expected {
		t.Errorf("%s: got %s, want %s", key, actual, expected)
	}
}

func TestViewerDrillDown(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), visible_levels_js+interaction_js+`
  mouse(document.querySelector('#layer-context [onclick="navigateDown()"]'), 'click');
  r.afterClick = visible();
  showLevel('code');
  navigateDown();
  r.afterLastLevel = visible();`)

	expect_json(t, actual, "afterClick", `["container"]`)
	expect_json(t, actual, "afterLastLevel", `["code"]`)
}

func TestViewerSizingModes(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), `
  var diagramWidth = function (level) {
    return Number(document.querySelector('#diagram-' + level + ' svg').getAttribute('width'));
  };
  r.nativeWidth = diagramWidth('context');
  r.recordedWidth = diagramData.context.width;
  toggleFitMode();
  r.fitText = document.getElementById('fit-text').textContent.trim();
  r.fitWidth = diagramWidth('context');
  r.fitsViewport = diagramWidth('context') <= window.innerWidth;
  showLevel('container');
  resizeContainers();
  r.stillFit = fitToWidth && diagramWidth('container') <= window.innerWidth;
  toggleFitMode();
  r.backText = document.getElementById('fit-text').textContent.trim();
  r.backWidth = diagramWidth('container');
  r.recordedContainerWidth = diagramData.container.width;`)

	if actual["nativeWidth"] != actual["recordedWidth"] {
		t.Errorf("native mode: diagram width %v, recorded %v", actual["nativeWidth"], actual["recordedWidth"])
	}
	expect_json(t, actual, "fitText", `"Auto Scale"`)
	expect_json(t, actual, "fitsViewport", `true`)
	expect_json(t, actual, "stillFit", `true`)
	expect_json(t, actual, "backText", `"Native Size"`)
	if actual["backWidth"] != actual["recordedContainerWidth"] {
		t.Errorf("back in native mode: diagram width %v, recorded %v", actual["backWidth"], actual["recordedContainerWidth"])
	}
}

func TestViewerNoteToggling(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), `
  var hidden = function (sel) {
    return Array.from(document.querySelectorAll(sel)).filter(function (e) { return e.style.display === 'none'; }).length;
  };
  r.notes = document.querySelectorAll('g.note').length;
  r.noteLinks = document.querySelectorAll('g.note-link').length;
  r.layersWithNotes = new Set(Array.from(document.querySelectorAll('g.note')).map(function (n) { return n.closest('g[id^="layer-"]').id; })).size;
  document.getElementById('notes-toggle').dispatchEvent(new MouseEvent('click', { bubbles: true }));
  r.hiddenNotes = hidden('g.note');
  r.hiddenNoteLinks = hidden('g.note-link');
  r.text = document.getElementById('notes-text').textContent.trim();
  toggleNotes();
  r.hiddenAfterShow = hidden('g.note') + hidden('g.note-link');
  r.textAfterShow = document.getElementById('notes-text').textContent.trim();`)

	expect_json(t, actual, "notes", `2`)
	expect_json(t, actual, "noteLinks", `2`)
	expect_json(t, actual, "layersWithNotes", `2`)
	expect_json(t, actual, "hiddenNotes", `2`)
	expect_json(t, actual, "hiddenNoteLinks", `2`)
	expect_json(t, actual, "text", `"Show Notes"`)
	expect_json(t, actual, "hiddenAfterShow", `0`)
	expect_json(t, actual, "textAfterShow", `"Hide Notes"`)
}

func TestViewerPathHighlighting(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), interaction_js+`
  var hitbox = hitboxes('context')[0];
  var link = hitbox.closest('g.link');
  var bg = link.querySelector('.text-bg');
  r.titles = document.querySelectorAll('#layer-context title').length;
  mouse(hitbox, 'mouseenter');
  r.hoverHighlighted = link.classList.contains('highlighted');
  r.hoverOpacity = bg.getAttribute('fill-opacity');
  r.hoverOnTop = link.parentNode.lastElementChild === link;
  mouse(hitbox, 'mouseleave');
  r.leaveHighlighted = link.classList.contains('highlighted');
  r.leaveOpacity = bg.getAttribute('fill-opacity');`)

	expect_json(t, actual, "titles", `0`)
	expect_json(t, actual, "hoverHighlighted", `true`)
	expect_json(t, actual, "hoverOpacity", `"0.9"`)
	expect_json(t, actual, "hoverOnTop", `true`)
	expect_json(t, actual, "leaveHighlighted", `false`)
	expect_json(t, actual, "leaveOpacity", `"0"`)
}

func TestViewerPinnedSelection(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), interaction_js+`
  showLevel('component');
  var boxes = hitboxes('component');
  r.labelledLinks = boxes.length;
  var a = boxes[0], b = boxes[1];
  mouse(a, 'click');
  r.afterPinA = pinned();
  mouse(a, 'mouseleave');
  r.staysPinnedAfterLeave = pinned();
  mouse(b, 'click');
  r.afterPinB = pinned();
  mouse(a, 'click', { ctrlKey: true });
  r.afterCtrlA = pinned();
  mouse(a, 'click', { ctrlKey: true });
  r.afterCtrlAAgain = pinned();
  mouse(b, 'click');
  r.afterClickPinnedB = pinned();
  mouse(a, 'click');
  mouse(b, 'click', { metaKey: true });
  key('Escape');
  r.afterEscape = pinned();`)

	expect_json(t, actual, "labelledLinks", `2`)
	expect_json(t, actual, "afterPinA", `["lnk1"]`)
	expect_json(t, actual, "staysPinnedAfterLeave", `["lnk1"]`)
	expect_json(t, actual, "afterPinB", `["lnk2"]`)
	expect_json(t, actual, "afterCtrlA", `["lnk1","lnk2"]`)
	expect_json(t, actual, "afterCtrlAAgain", `["lnk2"]`)
	expect_json(t, actual, "afterClickPinnedB", `[]`)
	expect_json(t, actual, "afterEscape", `[]`)
}

// Escape clears pins on every layer after any number of level switches
func TestViewerEscapeClearsAcrossLayers(t *testing.T) {
	actual := run_viewer(t, filepath.Join("testdata", "notes-and-images"), interaction_js+`
  for (var i = 0; i < 6; i++) {
    showLevel(availableLevels[i % availableLevels.length]);
  }
  showLevel('context');
  mouse(hitboxes('context')[0], 'click', { ctrlKey: true });
  showLevel('container');
  mouse(hitboxes('container')[0], 'click', { ctrlKey: true });
  r.beforeEscape = pinned();
  key('Escape');
  r.afterEscape = pinned();`)

	expect_json(t, actual, "beforeEscape", `["link_web_app_database","lnk5"]`)
	expect_json(t, actual, "afterEscape", `[]`)
}
