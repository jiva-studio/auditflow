// Package rules implements the rules provider adapter for loading rules from file.
package rules

import (
	"errors"
	"fmt"
	"strings"

	domainRules "assessment/libs/domain/rules"
	"assessment/libs/rules"
	"assessment/modules/agent/internal/ports"
)

// Sentinel errors for FileRulesProvider.
var (
	ErrEmptyRulesPath = errors.New("rules path cannot be empty")
)

// FileRulesProvider loads business rules from a specified JSON file.
type FileRulesProvider struct {
	rules []domainRules.Rule
}

// NewFileRulesProvider loads rules from the file at rulesPath.
func NewFileRulesProvider(rulesPath string) (*FileRulesProvider, error) {
	if strings.TrimSpace(rulesPath) == "" {
		return nil, ErrEmptyRulesPath
	}

	loadedRules, err := rules.LoadRulesFromFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("load rules from %s: %w", rulesPath, err)
	}

	return &FileRulesProvider{
		rules: loadedRules,
	}, nil
}

var _ ports.RulesProvider = (*FileRulesProvider)(nil)

// GetRules returns the loaded slice of domain rules.
func (p *FileRulesProvider) GetRules() []domainRules.Rule {
	res := make([]domainRules.Rule, len(p.rules))
	copy(res, p.rules)
	return res
}
