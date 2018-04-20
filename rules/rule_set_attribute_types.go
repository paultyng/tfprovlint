package rules

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"github.com/paultyng/tfprovlint/ssahelp"
)

var allowedBasicKind = map[provparse.AttributeType][]types.BasicKind{
	provparse.TypeBool: []types.BasicKind{types.Bool},
	provparse.TypeInt: []types.BasicKind{
		types.Int,
		types.Int16,
		types.Int32,
		types.Int64,
		types.Int8,
		// TODO: should uints be allowed? not sure??
		// types.Uint,
		// types.Uint16,
		// types.Uint32,
		// types.Uint64,
		// types.Uint8,
	},
	provparse.TypeFloat: []types.BasicKind{
		types.Float32,
		types.Float64,
	},
	provparse.TypeString: []types.BasicKind{
		types.String,
	},
}

func NewUseProperAttributeTypesInSetRule() lint.ResourceRule {
	r := &resourceDataSetRule{}

	// TODO: probably should just make this an interface thing if i have to pass the instance anyway
	r.CheckAttributeSet = useProperAttributeTypesInSet(r)

	return r
}

func useProperAttributeTypesInSet(rule *resourceDataSetRule) func(r *provparse.Resource, att *provparse.Attribute, attName string, ssacall ssa.CallInstruction) ([]lint.Issue, error) {
	return func(r *provparse.Resource, att *provparse.Attribute, attName string, ssacall ssa.CallInstruction) ([]lint.Issue, error) {
		if att == nil {
			// no matched attribute, skip
			return nil, nil
		}

		var wrongType = func() ([]lint.Issue, error) {
			return []lint.Issue{
				{
					Pos:     ssacall.Pos(),
					Message: fmt.Sprintf("attribute %q expects a d.Set compatible with %v", attName, att.Type),
				},
			}, nil
		}

		argValue := ssacall.Common().Args[2]
		argValue = ssahelp.RootValue(argValue)

		// TODO: if this is a call to Get of an attribute of the same name, this is ok
		// see the vsphere provider: github.com/terraform-providers/terraform-provider-vsphere/vsphere/resource_vsphere_virtual_disk.go

		t := argValue.Type()
		t = ssahelp.DerefType(t)
		if named, ok := t.(*types.Named); ok {
			t = named.Underlying()
		}

		if kinds, ok := allowedBasicKind[att.Type]; ok {
			if basic, ok := t.(*types.Basic); ok {
				kindFound := false
				for _, k := range kinds {
					if basic.Kind() == k {
						kindFound = true
						break
					}
				}
				if !kindFound {
					// this is a basic but the wrong one
					return wrongType()
				}
			} else {
				// not a basic type when attribute expects it
				// TODO: trace output
				return wrongType()
			}
			return nil, nil
		}

		rule.warnf("complex set type checking not yet implemented")

		// TODO: fill this in!
		switch att.Type {
		case provparse.TypeList:
		case provparse.TypeSet:
		case provparse.TypeMap:
		}

		return nil, nil
	}
}
