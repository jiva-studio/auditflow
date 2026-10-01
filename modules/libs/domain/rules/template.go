package rules

import (
	"errors"
	"strings"
)

// ErrEmptyTemplateTitle is returned when a popup template has an empty title.
var ErrEmptyTemplateTitle = errors.New("template title cannot be empty")

// PopupTemplate represents title and body templates with placeholder variables.
type PopupTemplate struct {
	Title string
	Body  string
}

// NewPopupTemplate creates and validates a PopupTemplate value object.
func NewPopupTemplate(title, body string) (PopupTemplate, error) {
	if strings.TrimSpace(title) == "" {
		return PopupTemplate{}, ErrEmptyTemplateTitle
	}
	return PopupTemplate{
		Title: title,
		Body:  body,
	}, nil
}

// Render replaces placeholder variables like {click}, {window_title}, {clipboard}, {ocr} in title and body
// using single-pass substitution to ensure deterministic output, prevent secondary expansion,
// and strip any unresolved placeholders.
func (t PopupTemplate) Render(vars map[string]string) (renderedTitle string, renderedBody string) {
	return renderString(t.Title, vars), renderString(t.Body, vars)
}

func renderString(tmpl string, vars map[string]string) string {
	var sb strings.Builder
	sb.Grow(len(tmpl))
	for i := 0; i < len(tmpl); {
		token, adv, ok := extractPlaceholder(tmpl[i:])
		if ok {
			if vars != nil {
				if val, exists := vars[token]; exists {
					sb.WriteString(val)
				}
			}
			i += adv
			continue
		}
		sb.WriteByte(tmpl[i])
		i++
	}
	return sb.String()
}

func extractPlaceholder(s string) (string, int, bool) {
	if len(s) < 2 || s[0] != '{' {
		return "", 0, false
	}
	closeIdx := strings.IndexByte(s, '}')
	if closeIdx == -1 {
		return "", 0, false
	}
	token := s[1:closeIdx]
	if len(token) == 0 || strings.ContainsAny(token, "{} \t\r\n") {
		return "", 0, false
	}
	return token, closeIdx + 1, true
}
