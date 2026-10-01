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
	return renderTemplate(t.Title, vars), renderTemplate(t.Body, vars)
}

func renderTemplate(tmpl string, vars map[string]string) string {
	if !strings.Contains(tmpl, "{") {
		return tmpl
	}

	var sb strings.Builder
	sb.Grow(len(tmpl))

	i := 0
	for i < len(tmpl) {
		start := strings.IndexByte(tmpl[i:], '{')
		if start == -1 {
			sb.WriteString(tmpl[i:])
			break
		}
		start += i
		sb.WriteString(tmpl[i:start])

		end := strings.IndexByte(tmpl[start+1:], '}')
		if end == -1 {
			sb.WriteString(tmpl[start:])
			break
		}
		end += start + 1

		// If there is an inner '{' between start+1 and end, write the prefix as literal
		if innerStart := strings.LastIndexByte(tmpl[start+1:end], '{'); innerStart != -1 {
			actualStart := start + 1 + innerStart
			sb.WriteString(tmpl[start:actualStart])
			start = actualStart
		}

		token := tmpl[start+1 : end]
		if vars != nil {
			if val, ok := vars[token]; ok {
				sb.WriteString(val)
				i = end + 1
				continue
			}
		}

		if token == "" {
			sb.WriteString("{}")
			i = end + 1
			continue
		}

		if strings.ContainsAny(token, " \t\r\n") {
			sb.WriteString(tmpl[start : end+1])
			i = end + 1
			continue
		}

		// Unresolved variable placeholder: cleanly strip
		i = end + 1
	}

	return sb.String()
}
