package rules

import "github.com/paultyng/tfprovlint/lint"

func NewNoSetIdInDeleteFuncRule() lint.ResourceRule {
	deleteBlacklist := map[string]bool{}
	deleteBlacklist[funcResourceDataSetId] = true

	return &callBlacklistRule{
		IssueMessageFormat: "DeleteFunc should not call %s",
		Delete:             deleteBlacklist,
	}
}
