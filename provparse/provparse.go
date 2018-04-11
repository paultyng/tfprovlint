package provparse

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/types"
	"os"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type provParser struct {
	prog *loader.Program
	pkg  *ssa.Package
}

// Package parses a provider package and returns the parsed data.
func Package(path string) (*Provider, error) {
	hadError := false

	conf := &loader.Config{
		Build:      &build.Default,
		ParserMode: parser.ParseComments,
		ImportPkgs: map[string]bool{},
		TypeChecker: types.Config{
			Error: func(err error) {
				// Only print the first error found
				if hadError {
					return
				}
				hadError = true
				fmt.Fprintln(os.Stderr, err)
			},
		},
	}
	conf.ImportPkgs[path] = true
	prog, err := conf.Load()
	if err != nil {
		return nil, err
	}
	ssaProg := ssautil.CreateProgram(prog, ssa.GlobalDebug)
	// build bodies of funcs
	ssaProg.Build()

	pkg := ssaProg.ImportedPackage(path)
	if pkg == nil {
		return nil, fmt.Errorf("provider package not found")
	}

	p := &provParser{
		prog: prog,
		pkg:  pkg,
	}

	//only one non-test package in the path
	prov, err := p.parse()
	if err != nil {
		return nil, unwrapError(err, prog.Fset)
	}
	return prov, nil
}
