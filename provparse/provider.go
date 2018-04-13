package provparse

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ssa"
)

const (
	pkgTFHelperSchema = "github.com/hashicorp/terraform/helper/schema"

	resourceStructTypeName = "github.com/hashicorp/terraform/helper/schema.Resource"
)

func importInfo(imp *ast.ImportSpec) (ident string, path string, err error) {
	path, err = strconv.Unquote(imp.Path.Value)
	if err != nil {
		return "", "", nil
	}
	if imp.Name != nil && imp.Name.Name != "" {
		ident = imp.Name.Name
	} else {
		ident = filepath.Base(path)
	}
	return
}

func (p *provParser) schemaFunc(name string) *ssa.Function {
	f := p.pkg.Func(name)
	if p.isSchemaFunc(f) {
		return f
	}
	return nil
}

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

func (p *provParser) isProviderFunc(f *ssa.Function) bool {
	return p.hasResultSelectorName(f, 0, "github.com/hashicorp/terraform/terraform", "ResourceProvider")
}

func (p *provParser) isSchemaFunc(f *ssa.Function) bool {
	return p.hasResultSelectorName(f, 0, pkgTFHelperSchema, "Schema")
}

func walkToKey(n ast.Node, key string) ast.Expr {
	var r ast.Expr
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.KeyValueExpr:
			if k, ok := n.Key.(*ast.Ident); ok && k.Name == key {
				r = n.Value
			}
		}

		return r == nil
	})
	return r
}

func walkToSchema(n ast.Node) (*ast.CompositeLit, error) {
	schemaAst := walkToKey(n, "Schema")
	if schemaAst == nil {
		return nil, nil
	}
	switch schemaAst := schemaAst.(type) {
	case *ast.CompositeLit:
		return schemaAst, nil
	case *ast.CallExpr:
		switch fAst := schemaAst.Fun.(type) {
		case *ast.SelectorExpr:
			log.Printf("[WARN] loading schema funcs from external packages is not supported yet %v.%s", fAst.X, fAst.Sel.Name)
			return nil, nil
		case *ast.Ident:
			log.Printf("[WARN] loading schema from local/global vars (%q) is not supported yet", fAst.Name)
			return nil, nil
		case *ast.FuncLit:
			log.Printf("[WARN] loading schema inline in a function literal is not supported")
			return nil, nil
		}
		return nil, nodeErrorf(schemaAst.Fun, "unexpected call type %T", schemaAst.Fun)
	case *ast.Ident:
		log.Printf("[WARN] loading schema from local/global vars (%q) is not supported yet", schemaAst.Name)
		return nil, nil
	}

	return nil, nodeErrorf(schemaAst, "unexpected schema node type %T", schemaAst)
}

func (p *provParser) lookupResourceFunc(f *ssa.Function, key string) (*ssa.Function, error) {
	store := findStructFieldStore(f, resourceStructTypeName, key)
	if store == nil {
		return nil, nil
	}
	v := rootValue(store.Val)
	// TODO: handle Noop and RemoveFromState
	resourceFunc, ok := v.(*ssa.Function)
	if !ok {
		return nil, nodeErrorf(v, "unable to determine function from value of type %T", v)
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
		Func: rf,

		CreateFunc: create,
		ReadFunc:   read,
		UpdateFunc: update,
		DeleteFunc: delete,
		ExistsFunc: exists,
	}

	schemaAst, err := walkToSchema(rf.Syntax())
	if err != nil {
		return nil, wrapNodeErrorf(err, rf, "error loading schema for %q", name)
	}
	attrs := []Attribute{}
	if schemaAst != nil {
		err = p.appendAttributes(&attrs, rf, schemaAst)
		if err != nil {
			return nil, fmt.Errorf("error with attributes for %q: %s", name, err)
		}

		sort.Slice(attrs, func(i, j int) bool {
			return attrs[i].Name < attrs[j].Name
		})
	}
	r.Attributes = attrs

	return r, nil
}

func (p *provParser) appendAttributes(attrs *[]Attribute, rf *ssa.Function, schemaAst *ast.CompositeLit) error {
	for _, e := range schemaAst.Elts {
		kv := e.(*ast.KeyValueExpr)

		var (
			k   string
			err error
		)

		if bl, ok := kv.Key.(*ast.BasicLit); !ok {
			return nodeErrorf(kv.Key, "expected a literal string key, got %T", kv.Key)
		} else {
			k, err = strconv.Unquote(bl.Value)
			if err != nil {
				return err
			}
		}

		switch v := kv.Value.(type) {
		case *ast.UnaryExpr:
			if v.Op != token.AND {
				return nodeErrorf(v, "unexpected unary operator %v", v.Op)
			}
			vlit := v.X.(*ast.CompositeLit)
			att, err := p.buildAttribute(rf, k, vlit)
			if err != nil {
				return err
			}

			*attrs = append(*attrs, att)
		case *ast.CompositeLit:
			att, err := p.buildAttribute(rf, k, v)
			if err != nil {
				return err
			}

			*attrs = append(*attrs, att)
		case *ast.CallExpr:
			callName := v.Fun.(*ast.Ident).Name
			sf := p.schemaFunc(callName)
			if sf == nil {
				return fmt.Errorf("unable to find schema func for %s", callName)
			}

			var childAst *ast.CompositeLit
			ast.Inspect(sf.Syntax(), func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CompositeLit:
					if sel, ok := n.Type.(*ast.SelectorExpr); ok {
						// TODO: check imported package as well
						if sel.Sel.Name != "Schema" {
							return true
						}

						childAst = n
					}
				}

				return childAst == nil
			})

			att, err := p.buildAttribute(sf, k, childAst)
			if err != nil {
				return err
			}

			*attrs = append(*attrs, att)
		case *ast.Ident:
			//local/global var
			// ignore for now, revisit this...
			log.Printf("[WARN] ignoring %s in %s, schema assigned from local/global variables are not yet supported", k, rf.Name())
		default:
			return nodeErrorf(v, "unexpected schema value node %T for %s", v, k)
		}
	}

	return nil
}

func (p *provParser) buildAttribute(sf *ssa.Function, name string, schema *ast.CompositeLit) (Attribute, error) {
	att := Attribute{
		Name:        name,
		Description: strings.TrimSpace(stringKeyValue(schema, "Description")),
		Required:    boolKeyValue(schema, "Required"),
		Optional:    boolKeyValue(schema, "Optional"),
		Computed:    boolKeyValue(schema, "Computed"),
		Type:        TypeInvalid,
	}

	st, err := keyValue(schema, "Type")
	if err != nil {
		return Attribute{}, err
	}

	switch st := st.(type) {
	case *ast.SelectorExpr:
		// TODO: check imported package
		switch st.Sel.Name {
		case "TypeBool":
			att.Type = TypeBool
		case "TypeInt":
			att.Type = TypeInt
		case "TypeFloat":
			att.Type = TypeFloat
		case "TypeString":
			att.Type = TypeString
		case "TypeList":
			att.Type = TypeList
		case "TypeMap":
			att.Type = TypeMap
		case "TypeSet":
			att.Type = TypeSet
		default:
			return Attribute{}, nodeErrorf(st, "unexpected type %q", st.Sel.Name)
		}
	default:
		return Attribute{}, fmt.Errorf("schema type ast of %T is unexpected", st)
	}

	// t, err := keyValue(schema, "Type")
	// if err != nil {
	// 	return Attribute{}, err
	// }

	// TODO: min/max handling

	childSchema, err := walkToSchema(schema)
	if err != nil {
		return Attribute{}, fmt.Errorf("unable to find schema for %q: %s", name, err)
	}
	if childSchema != nil {

		atts := []Attribute{}
		err := p.appendAttributes(&atts, sf, childSchema)
		if err != nil {
			return Attribute{}, err
		}

		att.Attributes = atts
	}

	return att, nil
}

func stringKeyValue(haystack *ast.CompositeLit, needle string) string {
	v, err := keyValue(haystack, needle)
	if err != nil {
		panic(err)
	}
	if v == nil {
		return ""
	}
	switch v := v.(type) {
	case *ast.BasicLit:
		s, err := strconv.Unquote(v.Value)
		if err != nil {
			panic(err)
		}
		return s
	default:
		panic(fmt.Sprintf("unexpected bool type %T", v))
	}
}

func boolKeyValue(haystack *ast.CompositeLit, needle string) bool {
	v, err := keyValue(haystack, needle)
	if err != nil {
		panic(err)
	}
	if v == nil {
		return false
	}
	switch v := v.(type) {
	case *ast.Ident:
		return v.Name == "true"
	default:
		panic(fmt.Sprintf("unexpected bool type %T", v))
	}
}

func keyValue(haystack *ast.CompositeLit, needle string) (ast.Expr, error) {
	for _, e := range haystack.Elts {
		kv := e.(*ast.KeyValueExpr)
		var (
			k   string
			err error
		)
		switch keyAst := kv.Key.(type) {
		case *ast.BasicLit:
			k, err = strconv.Unquote(keyAst.Value)
			if err != nil {
				return nil, err
			}
		case *ast.Ident:
			k = keyAst.Name
		default:
			return nil, fmt.Errorf("unexpected key type %T", keyAst)
		}

		if k == needle {
			return kv.Value, nil
		}
	}

	return nil, nil
}

func skipFirstLine(s string) string {
	parts := strings.SplitN(s, "\n", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
