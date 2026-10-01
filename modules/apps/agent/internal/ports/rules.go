package ports

import (
	"assessment/modules/libs/domain/rules"
)

// RulesProvider is the port for retrieving configured business rules.
type RulesProvider interface {
	GetRules() []rules.Rule
}
