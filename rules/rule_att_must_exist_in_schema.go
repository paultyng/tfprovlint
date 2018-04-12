package rules

import (
	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"golang.org/x/tools/go/ssa"
)

func NewSetAttributeNameExistsRule() lint.ResourceRule {
	return &resourceDataSet{
		CheckAttribute: setAttributeNameExists,
	}
}

func setAttributeNameExists(r *provparse.Resource, att *provparse.Attribute, attName string, ssacall ssa.CallInstruction) ([]lint.Issue, error) {
	if att == nil {
		return []lint.Issue{
			lint.NewIssuef(ssacall.Pos(), "attribute %q was not read from the schema", attName),
		}, nil
	}
	return nil, nil
}
