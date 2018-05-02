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
	Create             map[string]bool
	Delete             map[string]bool
	Exists             map[string]bool
	Read               map[string]bool
	Update             map[string]bool
}

var _ lint.ResourceRule = &callBlacklistRule{}

func (rule *callBlacklistRule) CheckResource(readOnly bool, r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	// Prevent duplicate issues
	existingCallPos := map[string]map[token.Pos]bool{}

	if !readOnly && r.CreateFunc != nil {
		if calls := rule.functionCalls(r.CreateFunc, rule.Create); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, positions := range calls {
				for _, pos := range positions {
					if _, ok := existingCallPos[call]; !ok {
						existingCallPos[call] = map[token.Pos]bool{pos: true}
						continue
					}
					if existingCallPos[call][pos] {
						continue
					}
					existingCallPos[call][pos] = true
					issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
				}
			}
		}
	}

	if !readOnly && r.DeleteFunc != nil {
		// if this is `schema.RemoveFromState` ignore it
		if funcName := normalizeSSAFunctionString(r.DeleteFunc); funcName == funcRemoveFromState {
			return nil, nil
		}

		if calls := rule.functionCalls(r.DeleteFunc, rule.Delete); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, positions := range calls {
				for _, pos := range positions {
					if _, ok := existingCallPos[call]; !ok {
						existingCallPos[call] = map[token.Pos]bool{pos: true}
						continue
					}
					if existingCallPos[call][pos] {
						continue
					}
					existingCallPos[call][pos] = true
					issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
				}
			}
		}
	}

	if !readOnly && r.ExistsFunc != nil {
		if calls := rule.functionCalls(r.ExistsFunc, rule.Exists); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, positions := range calls {
				for _, pos := range positions {
					if _, ok := existingCallPos[call]; !ok {
						existingCallPos[call] = map[token.Pos]bool{pos: true}
						continue
					}
					if existingCallPos[call][pos] {
						continue
					}
					existingCallPos[call][pos] = true
					issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
				}
			}
		}
	}

	if !readOnly && r.ReadFunc != nil {
		if calls := rule.functionCalls(r.ReadFunc, rule.Read); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, positions := range calls {
				for _, pos := range positions {
					if _, ok := existingCallPos[call]; !ok {
						existingCallPos[call] = map[token.Pos]bool{pos: true}
						continue
					}
					if existingCallPos[call][pos] {
						continue
					}
					existingCallPos[call][pos] = true
					issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
				}
			}
		}
	}

	if !readOnly && r.UpdateFunc != nil {
		if calls := rule.functionCalls(r.UpdateFunc, rule.Update); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, positions := range calls {
				for _, pos := range positions {
					if _, ok := existingCallPos[call]; !ok {
						existingCallPos[call] = map[token.Pos]bool{pos: true}
						continue
					}
					if existingCallPos[call][pos] {
						continue
					}
					existingCallPos[call][pos] = true
					issues = append(issues, lint.NewIssuef(pos, rule.IssueMessageFormat, call))
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
