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

func loadRules(includes, excludes []string) map[string]ruleFactoryFunc {
	var filtered map[string]ruleFactoryFunc
	if len(includes) == 0 {
		filtered = resourceRules
	} else {
		filtered = make(map[string]ruleFactoryFunc, len(includes))
		for _, id := range includes {
			if rule, ok := resourceRules[id]; ok {
				filtered[id] = rule
			}
		}
	}
	if len(excludes) > 0 {
		for _, id := range excludes {
			if _, ok := filtered[id]; ok {
				delete(filtered, id)
			}
		}
	}

	return filtered
}
