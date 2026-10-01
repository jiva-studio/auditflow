// Package rules provides JSON loading, validation, and conversion into domain rules.
package rules

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	domain "assessment/modules/libs/domain/rules"
)

// LoadRules decodes and validates domain rules from an io.Reader.
func LoadRules(reader io.Reader) ([]domain.Rule, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules data: %w", err)
	}
	return LoadRulesFromBytes(data)
}

// LoadRulesFromFile reads, decodes and validates domain rules from a local file path.
func LoadRulesFromFile(filePath string) ([]domain.Rule, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rules file %q: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	return LoadRules(file)
}

// LoadRulesFromBytes decodes and validates domain rules from raw JSON bytes.
func LoadRulesFromBytes(data []byte) ([]domain.Rule, error) {
	var fileDTO RuleFileDTO
	if err := json.Unmarshal(data, &fileDTO); err != nil {
		return nil, fmt.Errorf("failed to parse rules JSON: %w", err)
	}

	result := make([]domain.Rule, 0, len(fileDTO.Rules))
	for i, ruleDTO := range fileDTO.Rules {
		rule, err := convertRuleDTO(ruleDTO)
		if err != nil {
			return nil, fmt.Errorf("invalid rule at index %d (%q): %w", i, ruleDTO.ID, err)
		}
		result = append(result, rule)
	}

	return result, nil
}

func convertRuleDTO(dto RuleDTO) (domain.Rule, error) {
	if strings.TrimSpace(dto.ID) == "" {
		return domain.Rule{}, ErrEmptyRuleID
	}
	if strings.TrimSpace(dto.Popup.Title) == "" {
		return domain.Rule{}, ErrMissingPopupTitle
	}

	popupTmpl, err := domain.NewPopupTemplate(dto.Popup.Title, dto.Popup.Body)
	if err != nil {
		return domain.Rule{}, err
	}

	when, err := convertWhenDTO(dto.When)
	if err != nil {
		return domain.Rule{}, err
	}

	return domain.NewRule(dto.ID, when, popupTmpl)
}

func convertWhenDTO(dto WhenDTO) (domain.WhenConditions, error) {
	when := domain.WhenConditions{}

	buildPatternList := func(sos StringOrSlice) (*domain.PatternList, error) {
		if len(sos) == 0 {
			return nil, nil
		}
		pl, err := domain.NewPatternList(sos...)
		if err != nil {
			return nil, err
		}
		return &pl, nil
	}

	var err error
	if when.Click, err = buildPatternList(dto.Click); err != nil {
		return domain.WhenConditions{}, err
	}
	if when.Process, err = buildPatternList(dto.Process); err != nil {
		return domain.WhenConditions{}, err
	}
	if when.WindowTitle, err = buildPatternList(dto.WindowTitle); err != nil {
		return domain.WhenConditions{}, err
	}
	if when.Clipboard, err = buildPatternList(dto.Clipboard); err != nil {
		return domain.WhenConditions{}, err
	}
	if when.OCR, err = buildPatternList(dto.OCR); err != nil {
		return domain.WhenConditions{}, err
	}

	return when, nil
}
