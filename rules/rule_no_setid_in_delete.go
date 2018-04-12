package rules

import "github.com/paultyng/tfprovlint/lint"

func NewNoSetIdInDeleteFuncRule() lint.ResourceRule {
	deleteBlacklist := map[string]bool{}
	deleteBlacklist[calleeResourceDataSetId] = true

	return &callBlacklist{
		IssueMessageFormat: "DeleteFunc should not call %s",
		Delete:             deleteBlacklist,
	}
}
