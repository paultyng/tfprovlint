package rules

import (
	"fmt"
	"log"
	"strconv"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
)

const (
	setCallee = "(*github.com/hashicorp/terraform/helper/schema.ResourceData).Set"
)

type resourceDataSet struct {
	CheckAttribute func(*provparse.Resource, *provparse.Attribute, string, ssa.CallInstruction) ([]lint.Issue, error)
}

var _ lint.ResourceRule = &resourceDataSet{}

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
					nameArg = valueBeforeInterface(nameArg)
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
