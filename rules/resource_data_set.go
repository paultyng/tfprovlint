package rules

import (
	"fmt"
	"strconv"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"github.com/paultyng/tfprovlint/ssahelp"
)

type resourceDataSetRule struct {
	commonRule

	CheckAttributeSet func(*provparse.Resource, *provparse.Attribute, string, ssa.CallInstruction) ([]lint.Issue, error)
}

var _ lint.ResourceRule = &resourceDataSetRule{}

func (rule *resourceDataSetRule) checkResourceFunc(r *provparse.Resource, f *ssa.Function) ([]lint.Issue, error) {
	var issues []lint.Issue
	var inspectErr error
	ssahelp.InspectInstructions(ssahelp.FuncInstructions(f), func(ins ssa.Instruction) bool {
		ssacall, ok := ins.(ssa.CallInstruction)
		if !ok {
			return true
		}

		if callee := ssacall.Common().StaticCallee(); callee != nil {
			calleeName := normalizeSSAFunctionString(callee)

			if calleeName == funcResourceDataSet {
				// look at the set!
				if len(ssacall.Common().Args) != 3 {
					inspectErr = fmt.Errorf("incorrect args count for ResourceData.Set call: %d", len(ssacall.Common().Args))
					return false
				}

				nameArg := ssacall.Common().Args[1]
				nameArg = ssahelp.RootValue(nameArg)
				var attName string
				switch nameArg := nameArg.(type) {
				case *ssa.Const:
					var err error
					attName, err = strconv.Unquote(nameArg.Value.ExactString())
					if err != nil {
						inspectErr = err
						return false
					}
				}
				if attName == "" {
					rule.warnf("unable to determine what attribute is being set in %s", r.ReadFunc.Name())
					return true
				}
				att := r.Attribute(attName)
				newIssues, err := rule.CheckAttributeSet(r, att, attName, ssacall)
				if err != nil {
					inspectErr = err
					return false
				}
				issues = append(issues, newIssues...)
			}
		}

		return true
	})
	if inspectErr != nil {
		return nil, inspectErr
	}
	return issues, nil
}

func (rule *resourceDataSetRule) CheckResource(readOnly bool, r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	if r.ReadFunc != nil {
		newIssues, err := rule.checkResourceFunc(r, r.ReadFunc)
		if err != nil {
			return nil, err
		}
		issues = append(issues, newIssues...)
	}

	// sets are mainly done in reads, but we  also  check creates and updates
	if r.CreateFunc != nil {
		newIssues, err := rule.checkResourceFunc(r, r.CreateFunc)
		if err != nil {
			return nil, err
		}
		issues = append(issues, newIssues...)
	}

	if r.UpdateFunc != nil {
		newIssues, err := rule.checkResourceFunc(r, r.UpdateFunc)
		if err != nil {
			return nil, err
		}
		issues = append(issues, newIssues...)
	}

	return issues, nil
}
