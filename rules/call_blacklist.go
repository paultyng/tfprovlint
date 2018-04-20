package rules

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"github.com/paultyng/tfprovlint/ssahelp"
)

type callBlacklistRule struct {
	commonRule

	IssueMessageFormat string
	RuleID             string
	Delete             map[string]bool
}

var _ lint.ResourceRule = &callBlacklistRule{}

func (rule *callBlacklistRule) CheckResource(r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	// TODO: start checking all the funcs?

	if r.DeleteFunc != nil {
		// if this is `schema.RemoveFromState` ignore it
		if funcName := normalizeSSAFunctionString(r.DeleteFunc); funcName == funcRemoveFromState {
			return nil, nil
		}

		if calls := rule.functionCalls(r.DeleteFunc, rule.Delete); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, pos := range calls {
				issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
			}
		}
	}

	return issues, nil
}

func (rule *callBlacklistRule) functionCalls(f *ssa.Function, callList map[string]bool) map[string]token.Pos {
	calls := map[string]token.Pos{}

	ssahelp.InspectInstructions(ssahelp.FuncInstructions(f), func(ins ssa.Instruction) bool {
		ssacall, ok := ins.(ssa.CallInstruction)
		if !ok {
			return true
		}

		if callee := ssacall.Common().StaticCallee(); callee != nil {
			calleeName := normalizeSSAFunctionString(callee)
			rule.tracef("checking %q against list", calleeName)
			if callList[calleeName] {
				calls[calleeName] = ssacall.Pos()
			}
		}

		return true
	})

	return calls
}
