package rules

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	domain "assessment/libs/domain/rules"
)

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error simulation")
}

// TestLoadRulesFromFile_AllDataRules tests all 5 rules from data/rules.json exhaustively.
func TestLoadRulesFromFile_AllDataRules(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "data", "rules.json")
	ruleList, err := LoadRulesFromFile(repoRoot)
	if err != nil {
		t.Fatalf("failed to load rules.json: %v", err)
	}

	if len(ruleList) != 5 {
		t.Fatalf("expected 5 rules in rules.json, got %d", len(ruleList))
	}

	rulesMap := make(map[string]domain.Rule, len(ruleList))
	for _, r := range ruleList {
		rulesMap[r.ID] = r
	}

	// 1. Rule: forwarded-email-opened ("click": "FW:*", "process": ["OUTLOOK.EXE", "olk.exe"])
	t.Run("Rule_forwarded_email_opened", func(t *testing.T) {
		r, ok := rulesMap["forwarded-email-opened"]
		if !ok {
			t.Fatal("rule forwarded-email-opened not found")
		}

		// Positive cases
		posCases := []domain.RuleEvaluationContext{
			{ClickText: "FW: Budget 2026", Process: "OUTLOOK.EXE"},
			{ClickText: "fw: urgent update", Process: "olk.exe"},
			{ClickText: "FW:RE: Test", Process: "Outlook.exe"},
		}
		for _, ctx := range posCases {
			title, body, matched := r.Evaluate(ctx)
			if !matched {
				t.Errorf("expected match for %+v", ctx)
			}
			if title != "Forwarded email" {
				t.Errorf("got title %q, want 'Forwarded email'", title)
			}
			expectedBody := fmt.Sprintf("You opened a forwarded email: %s", ctx.ClickText)
			if body != expectedBody {
				t.Errorf("got body %q, want %q", body, expectedBody)
			}
		}

		// Negative cases
		negCases := []domain.RuleEvaluationContext{
			{ClickText: "RE: Budget 2026", Process: "OUTLOOK.EXE"}, // wrong click prefix
			{ClickText: "FW: Budget 2026", Process: "chrome.exe"},  // wrong process
			{ClickText: "", Process: "OUTLOOK.EXE"},                // empty click
		}
		for _, ctx := range negCases {
			_, _, matched := r.Evaluate(ctx)
			if matched {
				t.Errorf("expected no match for %+v", ctx)
			}
		}
	})

	// 2. Rule: urgent-email-opened ("click": "*urgent*", "process": ["OUTLOOK.EXE", "olk.exe"])
	t.Run("Rule_urgent_email_opened", func(t *testing.T) {
		r, ok := rulesMap["urgent-email-opened"]
		if !ok {
			t.Fatal("rule urgent-email-opened not found")
		}

		// Positive cases
		posCases := []domain.RuleEvaluationContext{
			{ClickText: "URGENT: Review needed", Process: "OUTLOOK.EXE"},
			{ClickText: "This is urgent please read", Process: "olk.exe"},
			{ClickText: "urgent", Process: "OUTLOOK.EXE"},
		}
		for _, ctx := range posCases {
			title, body, matched := r.Evaluate(ctx)
			if !matched {
				t.Errorf("expected match for %+v", ctx)
			}
			if title != "Urgent email" || body != ctx.ClickText {
				t.Errorf("unexpected output: %q / %q", title, body)
			}
		}

		// Negative cases
		negCases := []domain.RuleEvaluationContext{
			{ClickText: "Important email", Process: "OUTLOOK.EXE"},
			{ClickText: "urgent email", Process: "notepad.exe"},
		}
		for _, ctx := range negCases {
			_, _, matched := r.Evaluate(ctx)
			if matched {
				t.Errorf("expected no match for %+v", ctx)
			}
		}
	})

	// 3. Rule: salesforce-record-deleted ("click": "Delete", "window_title": "*| Salesforce*", "ocr": "Are you sure you want to delete*")
	t.Run("Rule_salesforce_record_deleted", func(t *testing.T) {
		r, ok := rulesMap["salesforce-record-deleted"]
		if !ok {
			t.Fatal("rule salesforce-record-deleted not found")
		}

		posCtx := domain.RuleEvaluationContext{
			ClickText:   "Delete",
			WindowTitle: "0012345 | Case | Salesforce - Google Chrome",
			OCRScreenTexts: []string{
				"Header",
				"Are you sure you want to delete this record permanently?",
				"Cancel",
			},
		}
		title, body, matched := r.Evaluate(posCtx)
		if !matched {
			t.Errorf("expected match for salesforce deletion")
		}
		if title != "Record deleted" || body != "Deleted in Salesforce: 0012345 | Case | Salesforce - Google Chrome" {
			t.Errorf("unexpected output: %s / %s", title, body)
		}

		// Negatives
		negCases := []domain.RuleEvaluationContext{
			{ClickText: "Cancel", WindowTitle: posCtx.WindowTitle, OCRScreenTexts: posCtx.OCRScreenTexts},
			{ClickText: "Delete", WindowTitle: "Inbox - Gmail", OCRScreenTexts: posCtx.OCRScreenTexts},
			{ClickText: "Delete", WindowTitle: posCtx.WindowTitle, OCRScreenTexts: []string{"Are you sure you want to save?"}},
		}
		for _, ctx := range negCases {
			if _, _, m := r.Evaluate(ctx); m {
				t.Errorf("expected no match for %+v", ctx)
			}
		}
	})

	// 4. Rule: jira-ticket-done ("click": "Done", "window_title": "* - Jira - *", "process": null)
	t.Run("Rule_jira_ticket_done", func(t *testing.T) {
		r, ok := rulesMap["jira-ticket-done"]
		if !ok {
			t.Fatal("rule jira-ticket-done not found")
		}

		posCases := []domain.RuleEvaluationContext{
			{ClickText: "Done", WindowTitle: "PROJ-101 - Jira - Google Chrome", Process: "chrome.exe"},
			{ClickText: "done", WindowTitle: "BUG-202 - Jira - Firefox", Process: "firefox.exe"},
		}
		for _, ctx := range posCases {
			title, body, matched := r.Evaluate(ctx)
			if !matched {
				t.Errorf("expected match for %+v", ctx)
			}
			if title != "Ticket moved to Done" || body != ctx.WindowTitle {
				t.Errorf("unexpected output: %s / %s", title, body)
			}
		}

		// Negatives
		if _, _, m := r.Evaluate(domain.RuleEvaluationContext{ClickText: "In Progress", WindowTitle: "PROJ-101 - Jira - Google Chrome"}); m {
			t.Error("expected no match for In Progress click")
		}
		if _, _, m := r.Evaluate(domain.RuleEvaluationContext{ClickText: "Done", WindowTitle: "Dashboard - Confluence"}); m {
			t.Error("expected no match for Confluence window title")
		}
	})

	// 5. Rule: invoice-ready-to-attach ("clipboard": "INV-*", "process": ["OUTLOOK.EXE", "olk.exe"])
	t.Run("Rule_invoice_ready_to_attach", func(t *testing.T) {
		r, ok := rulesMap["invoice-ready-to-attach"]
		if !ok {
			t.Fatal("rule invoice-ready-to-attach not found")
		}

		posCtx := domain.RuleEvaluationContext{
			ClipboardText: "INV-998877",
			Process:       "OUTLOOK.EXE",
		}
		title, body, matched := r.Evaluate(posCtx)
		if !matched {
			t.Error("expected match for invoice clipboard")
		}
		if title != "Invoice in clipboard" || body != "You copied INV-998877. Attach the invoice to this email?" {
			t.Errorf("unexpected output: %s / %s", title, body)
		}

		// Negatives
		if _, _, m := r.Evaluate(domain.RuleEvaluationContext{ClipboardText: "DOC-998877", Process: "OUTLOOK.EXE"}); m {
			t.Error("expected no match for DOC- prefix")
		}
		if _, _, m := r.Evaluate(domain.RuleEvaluationContext{ClipboardText: "INV-998877", Process: "calc.exe"}); m {
			t.Error("expected no match for calc.exe process")
		}
	})
}

// TestLoadRulesFromBytes_ExhaustiveEdgeCases tests JSON decoding resilience.
func TestLoadRulesFromBytes_ExhaustiveEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty rules array",
			jsonInput: `{"rules": []}`,
			wantCount: 0,
		},
		{
			name:      "rule with all nulls in when",
			jsonInput: `{"rules": [{"id": "r1", "when": {"click": null, "process": null, "window_title": null, "clipboard": null, "ocr": null}, "popup": {"title": "T", "body": "B"}}]}`,
			wantCount: 1,
		},
		{
			name:      "rule with omitted when section",
			jsonInput: `{"rules": [{"id": "r1", "popup": {"title": "T", "body": "B"}}]}`,
			wantCount: 1,
		},
		{
			name:      "rule with empty body in popup",
			jsonInput: `{"rules": [{"id": "r1", "popup": {"title": "Title Only"}}]}`,
			wantCount: 1,
		},
		{
			name:      "rule with extra unexpected json fields (ignored gracefully)",
			jsonInput: `{"rules": [{"id": "r1", "description": "some extra", "priority": 10, "popup": {"title": "T"}}]}`,
			wantCount: 1,
		},
		{
			name:      "rule with unicode and cyrillic characters",
			jsonInput: `{"rules": [{"id": "правило-1", "when": {"click": "Удалить*", "window_title": "Отчет.xlsx*"}, "popup": {"title": "Внимание", "body": "Файл {window_title}"}}]}`,
			wantCount: 1,
		},
		{
			name:      "invalid json syntax - unclosed brace",
			jsonInput: `{"rules": [{"id": "r1"`,
			wantErr:   true,
		},
		{
			name:      "invalid json root - array instead of object",
			jsonInput: `[{"id": "r1"}]`,
			wantErr:   true,
		},
		{
			name:      "invalid json root - string",
			jsonInput: `"not a json object"`,
			wantErr:   true,
		},
		{
			name:      "invalid rule - empty id",
			jsonInput: `{"rules": [{"id": "", "popup": {"title": "T"}}]}`,
			wantErr:   true,
		},
		{
			name:      "invalid rule - whitespace only id",
			jsonInput: `{"rules": [{"id": "   \t\n  ", "popup": {"title": "T"}}]}`,
			wantErr:   true,
		},
		{
			name:      "invalid rule - empty popup title",
			jsonInput: `{"rules": [{"id": "r1", "popup": {"title": ""}}]}`,
			wantErr:   true,
		},
		{
			name:      "invalid rule - whitespace only popup title",
			jsonInput: `{"rules": [{"id": "r1", "popup": {"title": "   "}}]}`,
			wantErr:   true,
		},
		{
			name:      "invalid when field - number instead of string",
			jsonInput: `{"rules": [{"id": "r1", "when": {"click": 999}, "popup": {"title": "T"}}]}`,
			wantErr:   true,
		},
		{
			name:      "invalid when field - boolean instead of string",
			jsonInput: `{"rules": [{"id": "r1", "when": {"process": true}, "popup": {"title": "T"}}]}`,
			wantErr:   true,
		},
		{
			name:      "invalid when field - object instead of string or array",
			jsonInput: `{"rules": [{"id": "r1", "when": {"ocr": {"nested": "value"}}, "popup": {"title": "T"}}]}`,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rules, err := LoadRulesFromBytes([]byte(tc.jsonInput))
			if (err != nil) != tc.wantErr {
				t.Fatalf("LoadRulesFromBytes() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && len(rules) != tc.wantCount {
				t.Errorf("got %d rules, want %d", len(rules), tc.wantCount)
			}
		})
	}
}

// TestLoadRules_BOMHandling tests UTF-8 Byte Order Mark support.
func TestLoadRules_BOMHandling(t *testing.T) {
	bomJSON := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"rules": [{"id": "bom-rule", "popup": {"title": "BOM Title"}}]}`)...)
	rules, err := LoadRulesFromBytes(bytes.TrimPrefix(bomJSON, []byte{0xEF, 0xBB, 0xBF}))
	if err != nil {
		t.Fatalf("failed to decode BOM stripped JSON: %v", err)
	}
	if len(rules) != 1 || rules[0].ID != "bom-rule" {
		t.Fatalf("unexpected loaded rule: %+v", rules)
	}
}

// TestLoadRules_LargeRuleSet tests performance and correctness with a large rule set (1000 rules).
func TestLoadRules_LargeRuleSet(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(`{"rules": [`)
	for i := 0; i < 1000; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		fmt.Fprintf(&buf, `{"id": "rule-%d", "when": {"click": "Click%d", "process": ["proc-%d.exe"]}, "popup": {"title": "Title %d", "body": "Body %d"}}`, i, i, i, i, i)
	}
	buf.WriteString(`]}`)

	rules, err := LoadRulesFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to load large rule set: %v", err)
	}
	if len(rules) != 1000 {
		t.Fatalf("expected 1000 rules, got %d", len(rules))
	}

	// Verify rule 777 matches correctly
	r777 := rules[777]
	title, body, matched := r777.Evaluate(domain.RuleEvaluationContext{
		ClickText: "Click777",
		Process:   "proc-777.exe",
	})
	if !matched || title != "Title 777" || body != "Body 777" {
		t.Fatalf("rule 777 evaluation failed: matched=%v, %s / %s", matched, title, body)
	}
}

// TestLoadRules_IOErrors tests file and reader IO errors.
func TestLoadRules_IOErrors(t *testing.T) {
	// Reader error
	_, err := LoadRules(errReader{})
	if err == nil {
		t.Error("expected error from errReader, got nil")
	}

	// Non-existent file
	_, err = LoadRulesFromFile("non_existent_file_path_12345.json")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	// Valid temporary file
	tmpFile, err := os.CreateTemp("", "rules-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	validJSON := `{"rules": [{"id": "test-rule", "popup": {"title": "Hello", "body": "World"}}]}`
	if _, err := tmpFile.WriteString(validJSON); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	_ = tmpFile.Close()

	loaded, err := LoadRulesFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "test-rule" {
		t.Fatalf("unexpected loaded rules: %+v", loaded)
	}
}
