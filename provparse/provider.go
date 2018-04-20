package provparse

import (
	"fmt"
	"go/ast"
	"go/types"
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
	rt = ssahelp.DerefType(rt)
	switch rt := rt.(type) {
	case *types.Named:
		actualPkg := rt.Obj().Pkg().Path()
		actualType := rt.Obj().Name()

		return strings.HasSuffix(actualPkg, pack) && actualType == selector
	default:
		p.warnf("unexpected result type %T", rt)
	}
	return false
}

func (p *provParser) isResourceFunc(f *ssa.Function) bool {
	return p.hasResultSelectorName(f, 0, pkgTFHelperSchema, "Resource")
}

func (p *provParser) buildResource(name string, rf *ssa.Function) (*Resource, error) {
	r := &Resource{
		Name: name,

		pos: rf.Pos(),
	}

	retValue := ssahelp.ReturnValue(rf, 0)
	retValue = ssahelp.RootValue(retValue)
	refs := *retValue.Referrers()

	for field, set := range map[string]func(*ssa.Function){
		"Create": func(f *ssa.Function) { r.CreateFunc = f },
		"Read":   func(f *ssa.Function) { r.ReadFunc = f },
		"Update": func(f *ssa.Function) { r.UpdateFunc = f },
		"Delete": func(f *ssa.Function) { r.DeleteFunc = f },
		"Exists": func(f *ssa.Function) { r.ExistsFunc = f },
	} {
		f, err := ssahelp.StructFieldFuncValue(refs, resourceStructTypeName, field)
		if err != nil {
			switch {
			case ssahelp.IsNoFieldAddrFound(err):
				continue
			case ssahelp.IsNoExpectedValueFound(err):
				r.PartialParse = true
				// TODO: warn here or trace or something
				continue
			default:
				return nil, wrapNodeErrorf(err, rf, "unable to determine resource func %q", field)
			}
		}
		set(f)
	}

	schemaVal, err := ssahelp.StructFieldValue(refs, resourceStructTypeName, "Schema")
	if err != nil {
		if !ssahelp.IsNoFieldAddrFound(err) {
			return nil, wrapNodeErrorf(err, rf, "unable to find resource schema")
		}
		p.tracef("unable to find schema: %s", err.Error())
		r.PartialParse = true
		return r, nil
	}
	schemaVal = ssahelp.RootValue(schemaVal)
	attrs := []Attribute{}
	partial, err := p.appendAttributes(&attrs, schemaVal)
	if err != nil {
		return nil, wrapNodeErrorf(err, schemaVal, "error with attributes for %q", name)
	}
	if partial {
		r.PartialParse = true
	}
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Name < attrs[j].Name
	})
	r.Attributes = attrs

	return r, nil
}

func (p *provParser) appendAttributes(attrs *[]Attribute, schemaVal ssa.Value) (bool, error) {
	switch v := schemaVal.(type) {
	case *ssa.Alloc:
		allocType := v.Type()
		allocType = ssahelp.DerefType(allocType)
		switch {
		case ssahelp.TypeMatch(allocType, resourceStructTypeName):
			var err error
			schemaVal, err = ssahelp.StructFieldValue(*v.Referrers(), resourceStructTypeName, "Schema")
			if err != nil {
				switch {
				case ssahelp.IsNoExpectedValueFound(err):
					p.tracef("unexpected value type when searching for attributes: %s", err.Error())
					return false, nil
				case ssahelp.IsNoFieldAddrFound(err):
					fallthrough
				default:
					// this is a problem, no assignment found?
					return false, wrapNodeErrorf(err, v, "unable to find resource Schema field")
				}
			}
			schemaVal = ssahelp.RootValue(schemaVal)
		case ssahelp.TypeMatch(allocType, schemaStructTypeName):
			//this is single type Elem, just return
			return false, nil
		}
	case *ssa.MakeMap:
		//do nothing
	}

	makeMap, ok := schemaVal.(*ssa.MakeMap)
	if !ok {
		p.tracef("expected MakeMap but found %T: %v", schemaVal, schemaVal)
		return true, nil
	}

	refs := makeMap.Referrers()
	if refs == nil {
		p.tracef("no referrers on MakeMap")
		return true, nil
	}

	partial := false
	for _, ref := range *refs {
		mapUpdate, ok := ref.(*ssa.MapUpdate)
		if !ok {
			continue
		}
		keyVal := ssahelp.RootValue(mapUpdate.Key)
		var attName string
		switch keyVal := keyVal.(type) {
		case *ssa.Const:
			var err error
			attName, err = strconv.Unquote(keyVal.Value.ExactString())
			if err != nil {
				return false, wrapNodeErrorf(err, keyVal, "error unquoting key")
			}
		case *ssa.Next:
			p.tracef("ignoring iterator Next when looking up key, dynamic assignment")
			partial = true
			continue
		default:
			return false, nodeErrorf(keyVal, "unable to determine Schema key for %T", keyVal)
		}

		mapUpdateVal := ssahelp.RootValue(mapUpdate.Value)
		att, err := p.buildAttribute(attName, mapUpdateVal)
		if err != nil {
			return false, wrapNodeErrorf(err, mapUpdate, "unable to build attribute %q", attName)
		}
		*attrs = append(*attrs, att)
	}

	return partial, nil
}

func (p *provParser) buildAttribute(name string, v ssa.Value) (Attribute, error) {
	refs := *v.Referrers()
	att := Attribute{
		Name: name,
		Type: TypeInvalid,

		pos: v.Pos(),
	}
	if v, err := ssahelp.StructFieldStringValue(refs, schemaStructTypeName, "Description"); err != nil {
		switch {
		case ssahelp.IsNoFieldAddrFound(err):
			p.tracef("no description found")
		case ssahelp.IsNoExpectedValueFound(err):
			p.tracef("unexpected value found for %q Description: %s", name, err.Error())
			att.PartialParse = true
		default:
			return Attribute{}, wrapNodeErrorf(err, &att, "unable to determine description")
		}
	} else {
		att.Description = v
	}

	for field, set := range map[string]func(bool){
		"Required": func(v bool) { att.Required = v },
		"Computed": func(v bool) { att.Computed = v },
		"Optional": func(v bool) { att.Optional = v },
	} {
		v, err := ssahelp.StructFieldBoolValue(refs, schemaStructTypeName, field)
		if err != nil {
			switch {
			case ssahelp.IsNoFieldAddrFound(err):
				continue
			case ssahelp.IsNoExpectedValueFound(err):
				p.tracef("unexpected value found for %q %s: %s", name, field, err.Error())
				att.PartialParse = true
				continue
			default:
				return Attribute{}, wrapNodeErrorf(err, &att, "unable to determine bool value for %q", field)
			}
		}
		set(v)
	}

	typeVal, err := ssahelp.StructFieldValue(refs, schemaStructTypeName, "Type")
	if err != nil {
		switch {
		case ssahelp.IsNoFieldAddrFound(err):
			// weirdly couldn't find type here
			att.PartialParse = true
			att.Type = TypeNotParsed
		case ssahelp.IsNoExpectedValueFound(err):
			p.tracef("unexpected value found for %q Type: %s", name, err.Error())
			att.PartialParse = true
			att.Type = TypeNotParsed
		default:
			return Attribute{}, wrapNodeErrorf(err, v, "unable to extract Schema.Type for attribute %q", name)
		}
	} else {
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
	}

	childrenFieldName := "Schema"
	if att.Type == TypeList || att.Type == TypeSet {
		childrenFieldName = "Elem"
	}

	schemaVal, err := ssahelp.StructFieldValue(refs, schemaStructTypeName, childrenFieldName)
	if err != nil {
		if !ssahelp.IsNoFieldAddrFound(err) {
			return Attribute{}, wrapNodeErrorf(err, v, "error looking for children")
		}
		schemaVal = nil
	}

	if schemaVal != nil {
		schemaVal = ssahelp.RootValue(schemaVal)
		attrs := []Attribute{}
		partial, err := p.appendAttributes(&attrs, schemaVal)
		if err != nil {
			return Attribute{}, wrapNodeErrorf(err, schemaVal, "error with attributes for %q", name)
		}
		if partial {
			att.PartialParse = true
		}
		sort.Slice(attrs, func(i, j int) bool {
			return attrs[i].Name < attrs[j].Name
		})
		att.Attributes = attrs
	}

	return att, nil
}
