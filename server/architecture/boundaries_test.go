package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const repositoryModule = "github.com/oublie6/awesome-zero-platform/server/"

func productionGoFiles(t *testing.T) []string {
	t.Helper()
	root := filepath.Clean("..")
	var result []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		result = append(result, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPlatformModuleImportBoundaries(t *testing.T) {
	for _, path := range productionGoFiles(t) {
		rel, err := filepath.Rel("..", path)
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 2 {
			continue
		}
		tree := parts[0]
		if tree != "foundation" && tree != "platform" && tree != "business" {
			continue
		}
		src, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range src.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(importPath, repositoryModule) {
				continue
			}
			dep := strings.Split(strings.TrimPrefix(importPath, repositoryModule), "/")
			if len(dep) == 0 {
				continue
			}
			if dep[0] == "apps" || tree == "foundation" && (dep[0] == "platform" || dep[0] == "business") ||
				tree == "platform" && dep[0] == "business" {
				t.Errorf("%s: %s must not depend on %s", rel, tree, importPath)
			}
			// gamecore is a game-specific shared kernel, not a dependency for
			// unrelated future business modules.
			if tree == "business" && dep[0] == "business" && len(parts) >= 2 && len(dep) >= 2 &&
				parts[1] != dep[1] && !(parts[1] == "doudizhu" && dep[1] == "gamecore") {
				t.Errorf("%s: business module %s imports another business module %s", rel, parts[1], dep[1])
			}
			if tree == "business" && dep[0] == "platform" && strings.Contains(importPath, "/adapter/") {
				t.Errorf("%s: business imports platform implementation %s", rel, importPath)
			}
		}
	}
}

var createTable = regexp.MustCompile("(?im)^\\s*CREATE TABLE IF NOT EXISTS\\s+`?([a-z0-9_]+)`?")
var sqlReference = regexp.MustCompile("(?i)\\b(?:FROM|JOIN|INTO|UPDATE|TABLE)\\s+`?([a-z][a-z0-9_]*)`?")

func tableOwner(table string) string {
	switch {
	case strings.HasPrefix(table, "foundation_"):
		return "foundation"
	case strings.HasPrefix(table, "identity_"):
		return "identity"
	case strings.HasPrefix(table, "authorization_"):
		return "authz"
	case strings.HasPrefix(table, "platform_audit_"):
		return "admin"
	case strings.HasPrefix(table, "game_"):
		return "gamecore"
	case strings.HasPrefix(table, "doudizhu_"):
		return "doudizhu"
	default:
		return ""
	}
}

func TestAllSchemaTablesHaveDocumentedOwnership(t *testing.T) {
	schema, err := os.ReadFile(filepath.Join("..", "database", "schema", "current.sql"))
	if err != nil {
		t.Fatal(err)
	}
	matches := createTable.FindAllStringSubmatch(string(schema), -1)
	if len(matches) == 0 {
		t.Fatal("no schema tables discovered")
	}
	for _, match := range matches {
		if tableOwner(strings.ToLower(match[1])) == "" {
			t.Errorf("table %s has no declared module owner; update ownership rules and docs", match[1])
		}
	}
}

func TestBusinessSQLDoesNotReadOtherModulesTables(t *testing.T) {
	for _, path := range productionGoFiles(t) {
		rel, err := filepath.Rel("..", path)
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 3 || parts[0] != "business" {
			continue
		}
		module := parts[1]
		src, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(src, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Errorf("%s: unquote SQL string: %v", rel, err)
				return true
			}
			for _, match := range sqlReference.FindAllStringSubmatch(value, -1) {
				owner := tableOwner(strings.ToLower(match[1]))
				if owner != "" && owner != module {
					t.Errorf("%s: module %s directly accesses %s owned by %s", rel, module, match[1], owner)
				}
			}
			return true
		})
	}
}

func TestPlatformPackagesCannotQueryBusinessTables(t *testing.T) {
	for _, path := range productionGoFiles(t) {
		rel, err := filepath.Rel("..", path)
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 2 || parts[0] != "platform" {
			continue
		}
		src, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(src, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			for _, match := range sqlReference.FindAllStringSubmatch(value, -1) {
				owner := tableOwner(strings.ToLower(match[1]))
				if owner == "doudizhu" || owner == "gamecore" {
					t.Errorf("%s: platform must not query %s-owned table %s", rel, owner, match[1])
				}
			}
			return true
		})
	}
}
