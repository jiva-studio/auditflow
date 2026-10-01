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

func TestPatternList(t *testing.T) {
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
}
