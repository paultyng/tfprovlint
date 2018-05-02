package rules

import "github.com/paultyng/tfprovlint/lint"

func NewNoErrwrapWrapfInResourceFuncRule() lint.ResourceRule {
	funcBlacklist := map[string]bool{}
	funcBlacklist[funcErrwrapWrapf] = true

	return &callBlacklistRule{
		IssueMessageFormat: "Resource functions should call fmt.Errorf instead of %s",
		Create:             funcBlacklist,
		Delete:             funcBlacklist,
		Exists:             funcBlacklist,
		Read:               funcBlacklist,
		Update:             funcBlacklist,
	}
}
