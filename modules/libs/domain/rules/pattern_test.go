package rules

import (
	"testing"
)

func TestPattern(t *testing.T) {
	p, err := NewPattern("FW:*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !p.Matches("FW: Meeting tomorrow") {
		t.Fatal("expected match with prefix")
	}
	if !p.Matches("fw: lowercase prefix") {
		t.Fatal("expected case-insensitive match")
	}
	if p.Matches("RE: Meeting") {
		t.Fatal("should not match different prefix")
	}

	urgent, _ := NewPattern("*urgent*")
	if !urgent.Matches("This is URGENT please read") {
		t.Fatal("expected wildcard substring match")
	}
	if !urgent.Matches("urgent") {
		t.Fatal("expected exact word match")
	}
	if urgent.Matches("regular email") {
		t.Fatal("should not match")
	}
}

func TestPatternList_Matches(t *testing.T) {
	pl, err := NewPatternList("OUTLOOK.EXE", "olk.exe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !pl.Matches("OUTLOOK.EXE") {
		t.Fatal("expected match for OUTLOOK.EXE")
	}
	if !pl.Matches("olk.exe") {
		t.Fatal("expected match for olk.exe")
	}
	if !pl.Matches("outlook.exe") {
		t.Fatal("expected case-insensitive match")
	}
	if pl.Matches("chrome.exe") {
		t.Fatal("should not match chrome.exe")
	}
	if pl.IsEmpty() {
		t.Fatal("expected non-empty pattern list")
	}
}

func TestPatternList_EdgeCases(t *testing.T) {
	// Pattern.Raw
	p, _ := NewPattern("test*")
	if p.Raw() != "test*" {
		t.Fatalf("expected raw pattern test*, got %s", p.Raw())
	}

	// Empty PatternList
	emptyPL, err := NewPatternList()
	if err != nil {
		t.Fatalf("unexpected error for empty pattern list: %v", err)
	}
	if !emptyPL.IsEmpty() {
		t.Fatal("expected empty pattern list")
	}
	if emptyPL.Matches("anything") {
		t.Fatal("empty pattern list should not match anything")
	}

	// PatternList with empty items
	plWithEmpty, _ := NewPatternList("", "  ", "valid")
	if !plWithEmpty.Matches("valid") {
		t.Fatal("expected match for valid pattern in list")
	}
}

func TestPattern_WildcardMatchingCombinations(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		text      string
		wantMatch bool
	}{
		// Asterisk combinations
		{"catch-all asterisk", "*", "any string here", true},
		{"catch-all on empty string", "*", "", true},
		{"multiple consecutive asterisks", "***test***", "prefix-test-suffix", true},
		{"interleaved asterisks", "*a*b*c*", "1a2b3c4", true},
		{"interleaved asterisks mismatch", "*a*b*c*", "1a2c3b4", false},
		{"prefix asterisk", "*report.pdf", "financial_report.pdf", true},
		{"prefix asterisk mismatch", "*report.pdf", "financial_report.docx", false},
		{"suffix asterisk", "INV-*", "INV-12345", true},
		{"suffix asterisk mismatch", "INV-*", "DOC-12345", false},
		{"infix asterisk", "* - Jira - *", "PROJECT-99 - Jira - Google Chrome", true},
		{"infix asterisk mismatch", "* - Jira - *", "PROJECT-99 - Confluence - Google Chrome", false},

		// Question mark combinations
		{"single question mark", "file?.txt", "file1.txt", true},
		{"single question mark mismatch", "file?.txt", "file12.txt", false},
		{"multiple question marks", "INV-????", "INV-1234", true},
		{"multiple question marks mismatch length", "INV-????", "INV-123", false},
		{"mixed star and question mark", "*doc_??.*", "my_doc_01.pdf", true},

		// Special regex characters treated as raw literals
		{"pipe literal in pattern", "*| Salesforce*", "00123 | Salesforce - Chrome", true},
		{"backslash in path", `C:\Users\*`, `c:\users\alice\doc.txt`, true},
		{"plus and parens in pattern", `Account (+123)*`, `account (+123) details`, true},
		{"brackets in pattern", `[URGENT]*`, `[urgent] action required`, true},
		{"dots in pattern", `outlook.exe`, `OUTLOOK.EXE`, true},
		{"dots in pattern mismatch", `outlook.exe`, `OUTLOOK.EXE.BAK`, false},

		// Unicode & Cyrillic
		{"cyrillic match", `Удалить*`, `УДАЛИТЬ запись`, true},
		{"cyrillic mismatch", `Удалить*`, `Создать запись`, false},
		{"cyrillic question mark match", `фа?л.txt`, `файл.txt`, true},
		{"cyrillic question mark start", `?апись`, `Запись`, true},
		{"cyrillic multi question marks", `Отчёт_??_??`, `отчёт_01_10`, true},
		{"emoji with asterisk and question mark", `🚀*🎉?`, `🚀 blast off 🎉✨`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewPattern(tc.pattern)
			if err != nil {
				t.Fatalf("failed to create pattern %q: %v", tc.pattern, err)
			}
			matched := p.Matches(tc.text)
			if matched != tc.wantMatch {
				t.Errorf("Pattern(%q).Matches(%q) = %v, want %v", tc.pattern, tc.text, matched, tc.wantMatch)
			}
		})
	}
}
