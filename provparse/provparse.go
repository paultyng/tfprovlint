package provparse

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

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

		//only one package
		return parseProviderPackage(fset, pkg)
	}

	return nil, fmt.Errorf("provider package not found")
}
