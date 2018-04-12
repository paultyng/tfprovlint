package rules

import (
	"fmt"
	"go/token"
	"log"
	"strconv"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"golang.org/x/tools/go/ssa"
)

const (
	ruleIDSetNameMustExist    = "tfprovlint002"
	ruleIDUseProperTypesInSet = "tfprovlint003"
	// When using `d.Set` there is no need to check the error when scalar types are expected.
	ruleIDErrCheckComplexSets = "tfprovlint004"
	// When using `d.Set` should not dereference pointer types
	ruleIDDoNotDereferencePointers = "tfprovlint005"

	setCallee = "(*github.com/hashicorp/terraform/helper/schema.ResourceData).Set"
)

type resourceDataSet struct {
	CheckAttribute func(*provparse.Resource, *provparse.Attribute, string, ssa.CallInstruction) ([]lint.Issue, error)
}

var _ lint.ResourceRule = &resourceDataSet{}

func NewSetAttributeNameExistsRule() lint.ResourceRule {
	return &resourceDataSet{
		CheckAttribute: setAttributeNameExists,
	}
}

func NewDoNotDereferencePointersInSet() lint.ResourceRule {
	return &resourceDataSet{
		CheckAttribute: doNotDereferencePointersInSet,
	}
}

func doNotDereferencePointersInSet(r *provparse.Resource, att *provparse.Attribute, attName string, ssacall ssa.CallInstruction) ([]lint.Issue, error) {
	argValue := ssacall.Common().Args[2]
	var issues []lint.Issue

	inspectValue(argValue, func(v ssa.Value) bool {
		switch v := v.(type) {
		case *ssa.UnOp:
			if v.Op == token.MUL {
				// how many ptrs deep! should be even unless this is a field address dereference
				if numStars(v.X.Type())%2 != 0 {
					if _, ok := v.X.(*ssa.FieldAddr); !ok {
						issues = []lint.Issue{
							lint.NewIssuef(ruleIDDoNotDereferencePointers, ssacall.Pos(), "do not dereference value for attribute %q when calling d.Set", attName),
						}

						return false
					}
				}
			}
		}

		return true
	})

	return issues, nil
}

func setAttributeNameExists(r *provparse.Resource, att *provparse.Attribute, attName string, ssacall ssa.CallInstruction) ([]lint.Issue, error) {
	if att == nil {
		return []lint.Issue{
			lint.NewIssuef(ruleIDSetNameMustExist, ssacall.Pos(), "attribute %q was not read from the schema", attName),
		}, nil
	}
	return nil, nil
}

func (rule *resourceDataSet) CheckResource(r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	// sets are mainly done in reads, but we could also probably check creates and updates
	if r.ReadFunc != nil {
		var inspectErr error
		inspectInstructions(r.ReadFunc, func(ins ssa.Instruction) bool {
			ssacall, ok := ins.(ssa.CallInstruction)
			if !ok {
				return true
			}

			if callee := ssacall.Common().StaticCallee(); callee != nil {
				calleeName := normalizeSSAFunctionString(callee)

				if calleeName == setCallee {
					// look at the set!
					if len(ssacall.Common().Args) != 3 {
						inspectErr = fmt.Errorf("incorrect args count for ResourceData.Set call: %d", len(ssacall.Common().Args))
						return false
					}

					nameArg := ssacall.Common().Args[1]
					var attName string
					inspectValue(nameArg, func(v ssa.Value) bool {
						switch nameArg := nameArg.(type) {
						case *ssa.Const:
							var err error
							attName, err = strconv.Unquote(nameArg.Value.ExactString())
							if err != nil {
								inspectErr = err
								return false
							}
						}
						return true
					})
					if inspectErr != nil {
						return false
					}
					if attName == "" {
						log.Printf("[WARN] unable to determine what attribute is being set in %s", r.ReadFunc.Name())
						return true
					}
					att := r.Attribute(attName)
					newIssues, err := rule.CheckAttribute(r, att, attName, ssacall)
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
	}

	return issues, nil
}
