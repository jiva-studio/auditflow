package rules

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileRulesProvider(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		_, err := NewFileRulesProvider("   ")
		if !errors.Is(err, ErrEmptyRulesPath) {
			t.Errorf("expected ErrEmptyRulesPath, got %v", err)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := NewFileRulesProvider("/non/existent/path/rules.json")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
	})

	t.Run("valid rules file", func(t *testing.T) {
		tmpDir := t.TempDir()
		rulesFile := filepath.Join(tmpDir, "rules.json")
		content := `{
			"rules": [
				{
					"id": "test-rule-1",
					"when": {"click": "ClickMe*"},
					"popup": {"title": "Title", "body": "Body"}
				}
			]
		}`
		if err := os.WriteFile(rulesFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write tmp file: %v", err)
		}

		provider, err := NewFileRulesProvider(rulesFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		loaded := provider.GetRules()
		if len(loaded) != 1 || loaded[0].ID != "test-rule-1" {
			t.Errorf("unexpected loaded rules: %+v", loaded)
		}
	})
}
