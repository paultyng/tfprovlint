package provparse

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type provParser struct {
	fset *token.FileSet
	pkg  *ast.Package
}

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

func (p *provParser) fileFor(pos token.Pos) (*ast.File, error) {
	tFile := p.fset.File(pos)
	if tFile == nil {
		return nil, fmt.Errorf("unable to find file from position")
	}
	astFile := p.pkg.Files[tFile.Name()]
	if astFile == nil {
		return nil, fmt.Errorf("unable to find package file for %s", tFile.Name())
	}
	return astFile, nil
}

func (p *provParser) selectorImports(sel *ast.SelectorExpr, pkg string) (bool, error) {
	selID, ok := sel.X.(*ast.Ident)
	if !ok {
		return false, fmt.Errorf("unexpected selector %T, wanted *ast.Ident", sel.X)
	}
	file, err := p.fileFor(sel.Pos())
	if err != nil {
		return false, err
	}
	for _, imp := range file.Imports {
		impID, impPath, err := importInfo(imp)
		if err != nil {
			return false, err
		}
		if selID.Name == impID && pkg == impPath {
			return true, nil
		}
	}
	return false, nil
}

type schemaFunc struct {
	Func       *ast.FuncDecl
	CommentMap ast.CommentMap
}

func (p *provParser) parse() (*Provider, error) {
	resFuncs := map[string]schemaFunc{}
	schemaFuncs := map[string]schemaFunc{}
	var provFunc *ast.FuncDecl

	for _, file := range p.pkg.Files {
		cmap := ast.NewCommentMap(p.fset, file, file.Comments)

		for _, dec := range file.Decls {
			switch dec := dec.(type) {
			case *ast.FuncDecl:
				switch {
				case p.isResourceFunc(dec):
					resFuncs[dec.Name.Name] = schemaFunc{
						Func:       dec,
						CommentMap: cmap,
					}
				case p.isSchemaFunc(dec):
					schemaFuncs[dec.Name.Name] = schemaFunc{
						Func:       dec,
						CommentMap: cmap,
					}
				case p.isProviderFunc(dec):
					provFunc = dec
				}
			}
		}
	}

	if provFunc == nil {
		return nil, fmt.Errorf("unable to find Provider export func")
	}

	dataSourceFuncs, resourceFuncs, err := p.extractProviderData(provFunc)
	if err != nil {
		return nil, err
	}

	dataSources := make([]Resource, 0, len(dataSourceFuncs))

	for name, fName := range dataSourceFuncs {
		r, err := p.buildResource(name, resFuncs[fName], schemaFuncs)
		if err != nil {
			return nil, err
		}

		dataSources = append(dataSources, *r)
	}

	resources := make([]Resource, 0, len(resourceFuncs))

	for name, fName := range resourceFuncs {
		r, err := p.buildResource(name, resFuncs[fName], schemaFuncs)
		if err != nil {
			return nil, err
		}

		resources = append(resources, *r)
	}

	return &Provider{
		DataSources: dataSources,
		Resources:   resources,
	}, nil
}

func (p *provParser) extractProviderData(provFunc *ast.FuncDecl) (map[string]string, map[string]string, error) {
	var (
		dsAst *ast.CompositeLit
		rAst  *ast.CompositeLit
	)

	ast.Inspect(provFunc, func(n ast.Node) bool {
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
				if ok, _ := p.selectorImports(f, "github.com/hashicorp/terraform/helper/schema"); ok && f.Sel.Name == "DataSourceResourceShim" {
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

func (p *provParser) hasResultSelectorName(f *ast.FuncDecl, i int, pack, selector string) bool {
	if f.Type.Results == nil || len(f.Type.Results.List) <= i {
		return false
	}

	switch dec := f.Type.Results.List[i].Type.(type) {
	case *ast.StarExpr:
		if sel, ok := dec.X.(*ast.SelectorExpr); ok && sel.Sel.Name == selector {
			ok, _ := p.selectorImports(sel, pack)
			return ok
		}
	case *ast.SelectorExpr:
		if ok, _ := p.selectorImports(dec, pack); !ok {
			return false
		}

		return dec.Sel.Name == selector
	}
	return false
}

func (p *provParser) isResourceFunc(f *ast.FuncDecl) bool {
	return p.hasResultSelectorName(f, 0, "github.com/hashicorp/terraform/helper/schema", "Resource")
}

func (p *provParser) isProviderFunc(f *ast.FuncDecl) bool {
	return p.hasResultSelectorName(f, 0, "github.com/hashicorp/terraform/terraform", "ResourceProvider")
}

func (p *provParser) isSchemaFunc(f *ast.FuncDecl) bool {
	return p.hasResultSelectorName(f, 0, "github.com/hashicorp/terraform/helper/schema", "Schema")
}

func walkToSchema(n ast.Node) *ast.CompositeLit {
	var schemaAst *ast.CompositeLit
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.KeyValueExpr:
			if k, ok := n.Key.(*ast.Ident); ok && k.Name == "Schema" {
				schemaAst = n.Value.(*ast.CompositeLit)
			}
		}

		return schemaAst == nil
	})
	return schemaAst
}

func (p *provParser) buildResource(name string, rf schemaFunc, schemaFuncs map[string]schemaFunc) (*Resource, error) {
	r := &Resource{
		Name:             name,
		Provider:         "azurerm",
		NameSuffix:       name[8:len(name)],
		ShortDescription: "",
		Description:      strings.TrimSpace(skipFirstLine(rf.Func.Doc.Text())),
	}

	schemaAst := walkToSchema(rf.Func.Body)
	attrs := []Attribute{}
	err := p.appendAttributes(&attrs, rf, schemaAst, schemaFuncs)
	if err != nil {
		return nil, fmt.Errorf("error with attributes for %s: %s", name, err)
	}

	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Name < attrs[j].Name
	})
	r.Attributes = attrs

	return r, nil
}

func (p *provParser) appendAttributes(attrs *[]Attribute, rf schemaFunc, schemaAst *ast.CompositeLit, schemaFuncs map[string]schemaFunc) error {
	for _, e := range schemaAst.Elts {
		kv := e.(*ast.KeyValueExpr)

		k, err := strconv.Unquote(kv.Key.(*ast.BasicLit).Value)
		if err != nil {
			return err
		}

		switch v := kv.Value.(type) {
		case *ast.UnaryExpr:
			if v.Op != token.AND {
				return fmt.Errorf("unexpected unary operator %v", v.Op)
			}
			vlit := v.X.(*ast.CompositeLit)
			att, err := p.buildAttribute(rf, k, vlit, schemaFuncs)
			if err != nil {
				return err
			}

			*attrs = append(*attrs, att)
		case *ast.CompositeLit:
			att, err := p.buildAttribute(rf, k, v, schemaFuncs)
			if err != nil {
				return err
			}

			*attrs = append(*attrs, att)
		case *ast.CallExpr:
			callName := v.Fun.(*ast.Ident).Name
			sf, ok := schemaFuncs[callName]
			if !ok {
				return fmt.Errorf("unable to find schema func for %s", callName)
			}

			var childAst *ast.CompositeLit
			ast.Inspect(sf.Func, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CompositeLit:
					if sel, ok := n.Type.(*ast.SelectorExpr); ok {
						if ok, _ := p.selectorImports(sel, "github.com/hashicorp/terraform/helper/schema"); !ok {
							return true
						}

						if sel.Sel.Name != "Schema" {
							return true
						}

						childAst = n
					}
				}

				return childAst == nil
			})

			att, err := p.buildAttribute(sf, k, childAst, schemaFuncs)
			if err != nil {
				return err
			}

			*attrs = append(*attrs, att)
		default:
			return fmt.Errorf("unexpected schema value node %T for %s", v, k)
		}
	}

	return nil
}

func (p *provParser) buildAttribute(sf schemaFunc, name string, schema *ast.CompositeLit, schemaFuncs map[string]schemaFunc) (Attribute, error) {
	att := Attribute{
		Name:        name,
		Description: strings.TrimSpace(stringKeyValue(schema, "Description")),
		Required:    boolKeyValue(schema, "Required"),
		Optional:    boolKeyValue(schema, "Optional"),
		Computed:    boolKeyValue(schema, "Computed"),
	}

	// t, err := keyValue(schema, "Type")
	// if err != nil {
	// 	return Attribute{}, err
	// }

	// TODO: min/max handling

	childSchema := walkToSchema(schema)
	if childSchema != nil {
		atts := []Attribute{}
		err := p.appendAttributes(&atts, sf, childSchema, schemaFuncs)
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
