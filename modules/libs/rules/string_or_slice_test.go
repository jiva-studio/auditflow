package rules

import (
	"encoding/json"
	"testing"
)

func TestStringOrSlice_UnmarshalJSON_Valid(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantSlice []string
	}{
		{name: "single string", jsonInput: `"OUTLOOK.EXE"`, wantSlice: []string{"OUTLOOK.EXE"}},
		{name: "array of strings", jsonInput: `["OUTLOOK.EXE", "olk.exe"]`, wantSlice: []string{"OUTLOOK.EXE", "olk.exe"}},
		{name: "null value", jsonInput: `null`, wantSlice: nil},
		{name: "empty array", jsonInput: `[]`, wantSlice: []string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var sos StringOrSlice
			if err := json.Unmarshal([]byte(tc.jsonInput), &sos); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sos) != len(tc.wantSlice) {
				t.Fatalf("got length %d, want %d", len(sos), len(tc.wantSlice))
			}
			for i := range sos {
				if sos[i] != tc.wantSlice[i] {
					t.Errorf("got[%d] = %q, want %q", i, sos[i], tc.wantSlice[i])
				}
			}
		})
	}
}

func TestStringOrSlice_UnmarshalJSON_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
	}{
		{name: "invalid type number", jsonInput: `12345`},
		{name: "invalid type boolean", jsonInput: `true`},
		{name: "invalid type object", jsonInput: `{"key": "value"}`},
		{name: "invalid json syntax in array", jsonInput: `["unclosed`},
		{name: "invalid string escape sequence", jsonInput: `"invalid \u12"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var sos StringOrSlice
			if err := json.Unmarshal([]byte(tc.jsonInput), &sos); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}

	var emptySOS StringOrSlice
	if err := emptySOS.UnmarshalJSON([]byte("   ")); err != nil || emptySOS != nil {
		t.Errorf("expected nil slice for empty bytes, got %v, %v", emptySOS, err)
	}
}
