package ports

import (
	"assessment/libs/domain/rules"
)

// RulesProvider is the port for retrieving configured business rules.
type RulesProvider interface {
	GetRules() []rules.Rule
}
