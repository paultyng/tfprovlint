package rules

import (
	"go/token"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"github.com/paultyng/tfprovlint/ssahelp"
)

func NewDoNotDereferencePointersInSetRule() lint.ResourceRule {
	return &resourceDataSet{
		CheckAttributeSet: doNotDereferencePointersInSet,
	}
}

func doNotDereferencePointersInSet(r *provparse.Resource, att *provparse.Attribute, attName string, ssacall ssa.CallInstruction) ([]lint.Issue, error) {
	var issues []lint.Issue

	argValue := ssacall.Common().Args[2]
	argValue = ssahelp.RootValue(argValue)
	switch argValue := argValue.(type) {
	case *ssa.UnOp:
		if argValue.Op == token.MUL {
			// skip field and index addr derefs as they are pointer lookups (I think?)
			// although this feels like it would break down at some point?
			switch argValue.X.(type) {
			case *ssa.FieldAddr:
				return nil, nil
			case *ssa.IndexAddr:
				return nil, nil
			}

			if stars := numStars(argValue.X.Type()); stars > 0 {
				return []lint.Issue{
					lint.NewIssuef(ssacall.Pos(), "do not dereference value for attribute %q when calling d.Set", attName),
				}, nil
			}
		}
	}

	return issues, nil
}
