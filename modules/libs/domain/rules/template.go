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

// Render replaces placeholder variables like {click}, {window_title}, {clipboard}, {ocr} in title and body.
// Any remaining unresolved placeholders of the form {placeholder_name} are stripped.
func (t PopupTemplate) Render(vars map[string]string) (renderedTitle string, renderedBody string) {
	renderedTitle = t.Title
	renderedBody = t.Body

	for key, val := range vars {
		placeholder := "{" + key + "}"
		renderedTitle = strings.ReplaceAll(renderedTitle, placeholder, val)
		renderedBody = strings.ReplaceAll(renderedBody, placeholder, val)
	}

	renderedTitle = stripUnresolvedPlaceholders(renderedTitle)
	renderedBody = stripUnresolvedPlaceholders(renderedBody)

	return renderedTitle, renderedBody
}

func stripUnresolvedPlaceholders(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == '{' {
			closeIdx := strings.IndexByte(s[i:], '}')
			if closeIdx != -1 {
				token := s[i+1 : i+closeIdx]
				if len(token) > 0 && !strings.ContainsAny(token, "{} \t\r\n") {
					i += closeIdx + 1
					continue
				}
			}
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}
