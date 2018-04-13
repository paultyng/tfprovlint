package provparse

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/ssahelp"
)

const (
	pkgTFHelperSchema = "github.com/hashicorp/terraform/helper/schema"

	resourceStructTypeName = "github.com/hashicorp/terraform/helper/schema.Resource"
	schemaStructTypeName   = "github.com/hashicorp/terraform/helper/schema.Schema"
)

func (p *provParser) resourceFunc(name string) *ssa.Function {
	f := p.pkg.Func(name)
	if p.isResourceFunc(f) {
		return f
	}
	return nil
}

func (p *provParser) parse() (*Provider, error) {
	provFunc := p.pkg.Func("Provider")

	if provFunc == nil {
		return nil, fmt.Errorf("unable to find Provider export func")
	}

	dataSourceFuncs, resourceFuncs, err := p.extractProviderData(provFunc)
	if err != nil {
		return nil, err
	}

	dataSources := make([]Resource, 0, len(dataSourceFuncs))

	for name, fName := range dataSourceFuncs {
		r, err := p.buildResource(name, p.resourceFunc(fName))
		if err != nil {
			return nil, err
		}

		dataSources = append(dataSources, *r)
	}

	resources := make([]Resource, 0, len(resourceFuncs))

	for name, fName := range resourceFuncs {
		r, err := p.buildResource(name, p.resourceFunc(fName))
		if err != nil {
			return nil, err
		}

		resources = append(resources, *r)
	}

	return &Provider{
		DataSources: dataSources,
		Resources:   resources,
		Fset:        p.prog.Fset,

		pos: provFunc.Pos(),
	}, nil
}

func (p *provParser) extractProviderData(provFunc *ssa.Function) (map[string]string, map[string]string, error) {
	var (
		dsAst *ast.CompositeLit
		rAst  *ast.CompositeLit
	)

	ast.Inspect(provFunc.Syntax(), func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.KeyValueExpr:
			if k, ok := n.Key.(*ast.Ident); ok {
				switch k.Name {
				case "DataSourcesMap":
					dsAst = n.Value.(*ast.CompositeLit)
				case "ResourcesMap":
					rAst = n.Value.(*ast.CompositeLit)
				}
			}
		}

		return dsAst == nil || rAst == nil
	})

	dataSources, err := p.extractResourceFuncNames(dsAst)
	if err != nil {
		return nil, nil, err
	}

	resources, err := p.extractResourceFuncNames(rAst)
	if err != nil {
		return nil, nil, err
	}

	return dataSources, resources, nil
}

func (p *provParser) extractResourceFuncNames(cl *ast.CompositeLit) (map[string]string, error) {
	// if _, ok := cl.Type.(*ast.MapType); !ok {
	// 	return error?
	// }

	res := map[string]string{}

	for _, e := range cl.Elts {
		kv := e.(*ast.KeyValueExpr)
		k, err := strconv.Unquote(kv.Key.(*ast.BasicLit).Value)
		if err != nil {
			return nil, err
		}

		switch v := kv.Value.(type) {
		case *ast.CallExpr:
			switch f := v.Fun.(type) {
			case *ast.Ident:
				res[k] = f.Name
				continue
			case *ast.SelectorExpr:
				// TODO: check package this type is imported from
				if f.Sel.Name == "DataSourceResourceShim" {
					if shimCall, ok := v.Args[1].(*ast.CallExpr); ok {
						if shimFunc, ok := shimCall.Fun.(*ast.Ident); ok {
							res[k] = shimFunc.Name
							//TODO: indicate its a shim to a data source somewhere?
							continue
						}
					}
				}
			}
		}
		return nil, fmt.Errorf("unable to parse %s", k)
	}

	return res, nil
}

func (p *provParser) hasResultSelectorName(f *ssa.Function, i int, pack, selector string) bool {
	results := f.Signature.Results()

	if results == nil || results.Len() <= i {
		return false
	}

	rt := results.At(i).Type()
	if p, ok := rt.(*types.Pointer); ok {
		rt = p.Elem()
	}

	switch rt := rt.(type) {
	case *types.Named:
		actualPkg := rt.Obj().Pkg().Path()
		actualType := rt.Obj().Name()

		return strings.HasSuffix(actualPkg, pack) && actualType == selector
	default:
		log.Printf("unexpected result type %T", rt)
	}
	return false
}

func (p *provParser) isResourceFunc(f *ssa.Function) bool {
	return p.hasResultSelectorName(f, 0, pkgTFHelperSchema, "Resource")
}

func (p *provParser) lookupResourceFunc(f *ssa.Function, key string) (*ssa.Function, error) {
	v := ssahelp.StructFieldValue(ssahelp.FuncInstructions(f), resourceStructTypeName, key)
	if v == nil {
		return nil, nil
	}
	v = ssahelp.RootValue(v)
	// TODO: handle Noop and RemoveFromState
	resourceFunc, ok := v.(*ssa.Function)
	if !ok {
		return nil, nodeErrorf(f, "unable to determine function from value of type %T", v)
	}
	return resourceFunc, nil
}

func (p *provParser) buildResource(name string, rf *ssa.Function) (*Resource, error) {
	//rf.WriteTo(os.Stdout)

	create, err := p.lookupResourceFunc(rf, "Create")
	if err != nil {
		return nil, err
	}

	read, err := p.lookupResourceFunc(rf, "Read")
	if err != nil {
		return nil, err
	}

	update, err := p.lookupResourceFunc(rf, "Update")
	if err != nil {
		return nil, err
	}

	delete, err := p.lookupResourceFunc(rf, "Delete")
	if err != nil {
		return nil, err
	}

	exists, err := p.lookupResourceFunc(rf, "Exists")
	if err != nil {
		return nil, err
	}

	r := &Resource{
		Name: name,

		pos: rf.Pos(),

		CreateFunc: create,
		ReadFunc:   read,
		UpdateFunc: update,
		DeleteFunc: delete,
		ExistsFunc: exists,
	}

	schemaVal := ssahelp.StructFieldValue(ssahelp.FuncInstructions(rf), resourceStructTypeName, "Schema")
	if schemaVal == nil {
		// unable to find schema
		// TODO: log warning?
		r.PartialParse = true
		return r, nil
	}
	schemaVal = ssahelp.RootValue(schemaVal)

	attrs := []Attribute{}
	err = p.appendAttributes(&attrs, schemaVal)
	if err != nil {
		return nil, fmt.Errorf("error with attributes for %q: %s", name, err)
	}

	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Name < attrs[j].Name
	})
	r.Attributes = attrs

	return r, nil
}

func (p *provParser) appendAttributes(attrs *[]Attribute, schemaVal ssa.Value) error {
	makeMap, ok := schemaVal.(*ssa.MakeMap)
	if !ok {
		return nodeErrorf(schemaVal, "unable to find the MakeMap, found %T instead", schemaVal)
	}

	refs := makeMap.Referrers()
	if refs == nil {
		return nil
	}

	for _, ref := range *refs {
		mapUpdate, ok := ref.(*ssa.MapUpdate)
		if !ok {
			continue
		}

		keyVal := ssahelp.RootValue(mapUpdate.Key)
		cons, ok := keyVal.(*ssa.Const)
		if !ok {
			return nodeErrorf(keyVal, "unable to determine Schema key for %T", keyVal)
		}
		attName, err := strconv.Unquote(cons.Value.ExactString())
		if err != nil {
			return wrapNodeErrorf(err, cons, "error unquoting key")
		}

		mapUpdateVal := ssahelp.RootValue(mapUpdate.Value)
		att, err := p.buildAttribute(attName, mapUpdateVal)
		if err != nil {
			return wrapNodeErrorf(err, mapUpdate, "unable to build attribute %q", attName)
		}
		*attrs = append(*attrs, att)
	}

	return nil
}

func (p *provParser) buildAttribute(name string, v ssa.Value) (Attribute, error) {
	refs := *v.Referrers()
	att := Attribute{
		Name:        name,
		Description: strings.TrimSpace(ssahelp.StructFieldStringValue(refs, schemaStructTypeName, "Description")),
		Required:    ssahelp.StructFieldBoolValue(refs, schemaStructTypeName, "Required"),
		Optional:    ssahelp.StructFieldBoolValue(refs, schemaStructTypeName, "Optional"),
		Computed:    ssahelp.StructFieldBoolValue(refs, schemaStructTypeName, "Computed"),
		Type:        TypeInvalid,

		pos: v.Pos(),
	}

	typeVal := ssahelp.StructFieldValue(refs, schemaStructTypeName, "Type")
	if typeVal == nil {
		return Attribute{}, nodeErrorf(v, "unable to extract Schema.Type for attribute %q", name)
	}
	typeVal = ssahelp.RootValue(typeVal)
	cst, ok := typeVal.(*ssa.Const)
	if !ok {
		return Attribute{}, nodeErrorf(typeVal, "unable to find Type const %T", typeVal)
	}

	switch AttributeType(cst.Int64()) {
	case TypeBool:
		att.Type = TypeBool
	case TypeInt:
		att.Type = TypeInt
	case TypeFloat:
		att.Type = TypeFloat
	case TypeString:
		att.Type = TypeString
	case TypeList:
		att.Type = TypeList
	case TypeMap:
		att.Type = TypeMap
	case TypeSet:
		att.Type = TypeSet
	default:
		return Attribute{}, nodeErrorf(cst, "unexpected type %q", cst.Value.ExactString())
	}

	schemaVal := ssahelp.StructFieldValue(refs, schemaStructTypeName, "Schema")
	if schemaVal != nil {
		schemaVal = ssahelp.RootValue(schemaVal)

		attrs := []Attribute{}
		if err := p.appendAttributes(&attrs, schemaVal); err != nil {
			return Attribute{}, wrapNodeErrorf(err, schemaVal, "error with attributes for %q", name)
		}

		sort.Slice(attrs, func(i, j int) bool {
			return attrs[i].Name < attrs[j].Name
		})
		att.Attributes = attrs
	}

	return att, nil
}
