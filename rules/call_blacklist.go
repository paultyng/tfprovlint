package rules

import (
	"go/token"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"golang.org/x/tools/go/ssa"
)

const (
	ruleIDDeleteShouldNotCallSetId = "tfprovlint001"
)

const (
	calleeResourceDataSetId = "(*github.com/hashicorp/terraform/helper/schema.ResourceData).SetId"
)

type callBlacklist struct {
	IssueMessageFormat string
	RuleID             string
	Delete             map[string]bool
}

var _ lint.ResourceRule = &callBlacklist{}

func NewNoSetIdInDeleteFuncRule() lint.ResourceRule {
	deleteBlacklist := map[string]bool{}
	deleteBlacklist[calleeResourceDataSetId] = true

	return &callBlacklist{
		RuleID:             ruleIDDeleteShouldNotCallSetId,
		IssueMessageFormat: "DeleteFunc should not call %s",
		Delete:             deleteBlacklist,
	}
}

func (rule *callBlacklist) CheckResource(r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	if r.DeleteFunc != nil {
		if calls := functionCalls(r.DeleteFunc, rule.Delete); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, pos := range calls {
				issues = append(issues, lint.NewIssuef(rule.RuleID, pos, rule.IssueMessageFormat, call))
			}
		}
	}

	return issues, nil
}

func functionCalls(f *ssa.Function, callList map[string]bool) map[string]token.Pos {
	calls := map[string]token.Pos{}

	inspectInstructions(f, func(ins ssa.Instruction) bool {
		ssacall, ok := ins.(ssa.CallInstruction)
		if !ok {
			return true
		}

		if callee := ssacall.Common().StaticCallee(); callee != nil {
			calleeName := normalizeSSAFunctionString(callee)
			// log.Printf("[TRACE] checking %q against list", calleeName)
			if callList[calleeName] {
				calls[calleeName] = ssacall.Pos()
			}
		}

		return true
	})

	return calls
}
