// Package rules provides JSON loading, validation, and conversion into domain rules.
package rules

// RuleFileDTO represents the top-level structure of a rules configuration JSON file.
type RuleFileDTO struct {
	Rules []RuleDTO `json:"rules"`
}

// RuleDTO represents a single rule JSON record.
type RuleDTO struct {
	ID    string   `json:"id"`
	When  WhenDTO  `json:"when"`
	Popup PopupDTO `json:"popup"`
}

// WhenDTO contains pattern matching conditions.
type WhenDTO struct {
	Click       StringOrSlice `json:"click"`
	Process     StringOrSlice `json:"process"`
	WindowTitle StringOrSlice `json:"window_title"`
	Clipboard   StringOrSlice `json:"clipboard"`
	OCR         StringOrSlice `json:"ocr"`
}

// PopupDTO defines popup title and body templates.
type PopupDTO struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
