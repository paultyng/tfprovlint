package rules

import (
	"fmt"
	"go/types"
	"log"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"golang.org/x/tools/go/ssa"
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
	return &resourceDataSet{
		CheckAttribute: useProperAttributeTypesInSet,
	}
}

func useProperAttributeTypesInSet(r *provparse.Resource, att *provparse.Attribute, attName string, ssacall ssa.CallInstruction) ([]lint.Issue, error) {
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
	argValue = valueBeforeInterface(argValue)
	t := argValue.Type()

	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
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
			return wrongType()
		}
		return nil, nil
	}

	log.Printf("[WARN] complex set type checking not yet implemented")

	// TODO: fill this in!
	switch att.Type {
	case provparse.TypeList:
	case provparse.TypeSet:
	case provparse.TypeMap:
	}

	return nil, nil
}
