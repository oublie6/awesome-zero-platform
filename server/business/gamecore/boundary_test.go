package gamecore_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGamecoreProductionImportAndVocabularyBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{"doudizhu", "landlord", "joker", "card rank", "card suit", "three-seat"}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		encoded, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(encoded))
		for _, term := range banned {
			if strings.Contains(lower, term) {
				t.Fatalf("production file %s contains concrete-game term %q", name, term)
			}
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, encoded, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			first := strings.Split(path, "/")[0]
			if strings.Contains(first, ".") {
				t.Fatalf("production file %s imports non-standard package %q", name, path)
			}
		}
	}
}
