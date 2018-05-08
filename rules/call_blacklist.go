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

	Create map[string]bool
	Read   map[string]bool
	Exists map[string]bool
	Update map[string]bool
	Delete map[string]bool
}

var _ lint.ResourceRule = &callBlacklistRule{}

func (rule *callBlacklistRule) CheckResource(readOnly bool, r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	for _, t := range []struct {
		blacklist map[string]bool
		f         *ssa.Function
	}{
		{rule.Create, r.CreateFunc},
		{rule.Read, r.ReadFunc},
		{rule.Exists, r.ExistsFunc},
		{rule.Update, r.UpdateFunc},
		{rule.Delete, r.DeleteFunc},
	} {
		if t.f != nil {
			// if this is `schema.RemoveFromState` ignore it
			if funcName := normalizeSSAFunctionString(t.f); funcName == funcRemoveFromState {
				return nil, nil
			}

			if calls := rule.functionCalls(t.f, t.blacklist); len(calls) > 0 {
				// it makes some of the calls, need to append issues
				for call, positions := range calls {
					for _, pos := range positions {
						issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
					}
				}
			}
		}
	}

	return issues, nil
}

func (rule *callBlacklistRule) functionCalls(f *ssa.Function, callList map[string]bool) map[string][]token.Pos {
	calls := map[string][]token.Pos{}

	ssahelp.InspectInstructions(ssahelp.FuncInstructions(f), func(ins ssa.Instruction) bool {
		ssacall, ok := ins.(ssa.CallInstruction)
		if !ok {
			return true
		}

		if callee := ssacall.Common().StaticCallee(); callee != nil {
			calleeName := normalizeSSAFunctionString(callee)
			rule.tracef("checking %q against list", calleeName)
			if callList[calleeName] {
				calls[calleeName] = append(calls[calleeName], ssacall.Pos())
			}
		}

		return true
	})

	return calls
}
