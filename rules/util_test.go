package rules

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func stringSliceToSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

func mustMakeSamplePkg(src string) *ssa.Package {
	pkg, err := makeSamplePkg(src)
	if err != nil {
		panic(err)
	}
	return pkg
}

func makeSamplePkg(src string) (*ssa.Package, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	files := []*ast.File{f}

	tpkg := types.NewPackage("test", "")

	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset, tpkg, files, ssa.SanityCheckFunctions,
	)
	if err != nil {
		return nil, err
	}
	pkg.Build()
	return pkg, nil
}
