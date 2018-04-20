package provparse

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/types"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type provParser struct {
	prog *loader.Program
	pkg  *ssa.Package
}

var (
	shouldTrace = false
	shouldWarn  = false
)

// TODO: do this better!
func init() {
	switch os.Getenv("LOG_LVL") {
	case "TRACE":
		shouldTrace = true
		fallthrough
	case "WARN":
		shouldWarn = true
	}
}

func (*provParser) tracef(format string, args ...interface{}) {
	if shouldTrace {
		log.Printf("[TRACE] "+format, args...)
	}
}

func (*provParser) warnf(format string, args ...interface{}) {
	if shouldWarn {
		log.Printf("[WARN] "+format, args...)
	}
}

// Package parses a provider package and returns the parsed data.
func Package(path string) (*Provider, error) {
	potentialPaths := []string{path}

	name := filepath.Base(path)
	if suffix, ok := providerName(name); ok {
		// if this is a valid provider repo name (terraform-provider-x) extract the suffix
		name = suffix

		potentialPaths = []string{
			filepath.Join(path, "provider"),
			filepath.Join(path, name),
		}
	}

	var (
		prog       *loader.Program
		loadedPath string
	)
	for _, path := range potentialPaths {
		conf := &loader.Config{
			Build:      &build.Default,
			ParserMode: parser.ParseComments,
			ImportPkgs: map[string]bool{},
			TypeChecker: types.Config{
				Error: func(err error) {
					// this removes the default output of the TypeChecker error handling
				},
			},
		}
		conf.ImportPkgs[path] = true
		var err error
		prog, err = conf.Load()
		if err != nil {
			// this is gross, but the code just does an errors.New...
			if err.Error() == "no initial packages were loaded" {
				continue
			}
			return nil, err
		}
		loadedPath = path
		break
	}
	if prog == nil {
		return nil, fmt.Errorf("unable to determine provider package")
	}
	ssaProg := ssautil.CreateProgram(prog, ssa.GlobalDebug)
	// build bodies of funcs
	ssaProg.Build()

	pkg := ssaProg.ImportedPackage(loadedPath)
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
