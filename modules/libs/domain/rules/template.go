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
func (t PopupTemplate) Render(vars map[string]string) (renderedTitle string, renderedBody string) {
	renderedTitle = t.Title
	renderedBody = t.Body

	for key, val := range vars {
		placeholder := "{" + key + "}"
		renderedTitle = strings.ReplaceAll(renderedTitle, placeholder, val)
		renderedBody = strings.ReplaceAll(renderedBody, placeholder, val)
	}
	return renderedTitle, renderedBody
}
