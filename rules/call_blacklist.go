package rules

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
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

func (rule *callBlacklist) CheckResource(r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	if r.DeleteFunc != nil {
		if calls := functionCalls(r.DeleteFunc, rule.Delete); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, pos := range calls {
				issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
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
