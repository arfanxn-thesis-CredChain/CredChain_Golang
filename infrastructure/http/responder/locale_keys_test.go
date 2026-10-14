package responder

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadLocaleFile(t *testing.T, lang string) map[string]any {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	path := filepath.Join(repoRoot, "locales", lang+".json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(bytes, &raw); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return raw
}

func collectFlatKeys(m map[string]any, prefix string, out map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch vv := v.(type) {
		case string:
			out[key] = vv
		case map[string]any:
			collectFlatKeys(vv, key, out)
		}
	}
}

func TestLocaleFiles_ContainEveryMessageKey(t *testing.T) {
	en := loadLocaleFile(t, "en")
	id := loadLocaleFile(t, "id")

	flatEn := map[string]string{}
	flatId := map[string]string{}
	collectFlatKeys(en, "", flatEn)
	collectFlatKeys(id, "", flatId)

	for _, key := range CodeToMessageKey {
		_, okEn := flatEn[key]
		_, okId := flatId[key]
		assert.True(t, okEn, "key %q missing from en.json", key)
		assert.True(t, okId, "key %q missing from id.json", key)
	}
}

var autoInjectedKeys = map[string]bool{
	"field": true, "min": true, "max": true, "values": true,
}

var placeholderRegex = regexp.MustCompile(`\{\{\s*\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

func collectWithMetadataKeys(t *testing.T) map[string]bool {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	keys := map[string]bool{}
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "WithMetadata" {
				return true
			}
			if len(call.Args) < 1 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			keys[strings.Trim(lit.Value, `"`)] = true
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return keys
}

func TestLocaleTemplateKeys_BackedByCodeOrAutoInject(t *testing.T) {
	metaKeys := collectWithMetadataKeys(t)

	for _, lang := range []string{"en", "id"} {
		raw := loadLocaleFile(t, lang)
		flat := map[string]string{}
		collectFlatKeys(raw, "", flat)

		for k, v := range flat {
			matches := placeholderRegex.FindAllStringSubmatch(v, -1)
			for _, m := range matches {
				placeholder := m[1]
				if autoInjectedKeys[placeholder] {
					continue
				}
				if k == "mail_credential_greeting" && placeholder == "Name" {
					continue // CredentialNotificationWorker supplies Name via go-i18n TemplateData.
				}
				if metaKeys[placeholder] {
					continue
				}
				t.Errorf("locale %s key %q references {{.%s}} but no WithMetadata(%q,...) found",
					lang, k, placeholder, placeholder)
			}
		}
	}
}
