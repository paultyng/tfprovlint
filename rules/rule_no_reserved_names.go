package rules

import (
	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
)

// these are taken from https://github.com/hashicorp/terraform/blob/05291ab9822dd794b8ad61a902af832898618d99/config/loader_hcl.go#L20-L42

var reservedDataSourceFields = []string{
	"connection",
	"count",
	"depends_on",
	"lifecycle",
	"provider",
	"provisioner",
}

var reservedResourceFields = []string{
	"connection",
	"count",
	"depends_on",
	"id",
	"lifecycle",
	"provider",
	"provisioner",
}

var reservedProviderFields = []string{
	"alias",
	"version",
}

type noReservedNamesRule struct {
	commonRule
}

func NewNoReservedNamesRule() lint.ResourceRule {
	return &noReservedNamesRule{}
}

func (rule *noReservedNamesRule) CheckResource(readOnly bool, r *provparse.Resource) ([]lint.Issue, error) {
	fields := reservedResourceFields
	if readOnly {
		fields = reservedDataSourceFields
	}
	fieldMap := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldMap[f] = true
	}

	issues := make([]lint.Issue, 0)
	for _, att := range r.Attributes {
		if fieldMap[att.Name] {
			issues = append(issues, lint.NewIssuef(att.Pos(), "%q is a reserved attribute name", att.Name))
		}
	}
	return issues, nil
}
