package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// rewrites each case's `expected.svg` instead of comparing against it
var update = flag.Bool("update", false, "rewrite golden expectations")

// fixed generation time so golden output is reproducible
var golden_time = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// generates the stacked document for one `testdata/<case>/` directory
func generate_case(t *testing.T, case_dir string) string {
	t.Helper()
	diagrams, err := load(case_dir)
	if err != nil {
		t.Fatalf("load %s: %v", case_dir, err)
	}
	actual, err := stack("Golden "+filepath.Base(case_dir), golden_time, diagrams)
	if err != nil {
		t.Fatalf("stack %s: %v", case_dir, err)
	}
	return actual
}

func TestGolden(t *testing.T) {
	cases, err := filepath.Glob(filepath.Join("testdata", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no golden cases under testdata/")
	}
	for _, case_dir := range cases {
		t.Run(filepath.Base(case_dir), func(t *testing.T) {
			actual := generate_case(t, case_dir)
			if _, err := decode_tokens([]byte(actual)); err != nil {
				t.Fatalf("output is not well-formed XML: %v", err)
			}
			expected_path := filepath.Join(case_dir, "expected.svg")
			if *update {
				if err := os.WriteFile(expected_path, []byte(actual), 0644); err != nil {
					t.Fatal(err)
				}
				return
			}
			expected, err := os.ReadFile(expected_path)
			if err != nil {
				t.Fatalf("read expectation (run `go test -update` to create it): %v", err)
			}
			if string(expected) != actual {
				t.Errorf("output differs from %s; run `go test -update` and review the diff", expected_path)
			}
		})
	}
}

// the same inputs, title and time must give byte-identical output
func TestGoldenIsDeterministic(t *testing.T) {
	given := filepath.Join("testdata", "basic")
	first := generate_case(t, given)
	second := generate_case(t, given)
	if first != second {
		t.Error("two generations of the same input differ")
	}
}
