package cmd

import (
	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/rules"
)

type ruleFactoryFunc func() lint.ResourceRule

var resourceRules = map[string]ruleFactoryFunc{
	"tfprovlint001": rules.NewNoSetIdInDeleteFuncRule,
	"tfprovlint002": rules.NewSetAttributeNameExistsRule,
	"tfprovlint003": rules.NewUseProperAttributeTypesInSetRule,
	// "tfprovlint004": err check sets on complex types
	"tfprovlint005": rules.NewDoNotDereferencePointersInSetRule,
	"tfprovlint026": rules.NewNoReservedNamesRule,
}

func loadRules(only []string) map[string]ruleFactoryFunc {
	if len(only) == 0 {
		return resourceRules
	}

	filtered := make(map[string]ruleFactoryFunc, len(only))

	for _, id := range only {
		if rule, ok := resourceRules[id]; ok {
			filtered[id] = rule
		}
	}

	return filtered
}
