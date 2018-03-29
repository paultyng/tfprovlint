package provparse

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

// Package parses a provider package and returns the parsed data.
func Package(path string) (*Provider, error) {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	for pkgName, pkg := range pkgs {
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		p := &provParser{
			fset: fset,
			pkg:  pkg,
		}

		//only one non-test package in the path
		return p.parse()
	}

	return nil, fmt.Errorf("provider package not found")
}
