package rules

import (
	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
)

func NewSetAttributeNameExistsRule() lint.ResourceRule {
	return &resourceDataSet{
		CheckAttributeSet: setAttributeNameExists,
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
