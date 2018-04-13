package rules

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/paultyng/tfprovlint/lint"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func assertIssueMsg(t *testing.T, expectedMsg string, issues []lint.Issue) {
	t.Helper()

	if expectedMsg == "" {
		if len(issues) > 0 {
			t.Fatalf("expected no issues but found %d", len(issues))
		}
		return
	}
	if len(issues) != 1 {
		t.Fatalf("expected only a single issue to be found (not %d)", len(issues))
	}
	if msg := issues[0].Message; msg != expectedMsg {
		t.Fatalf("unexpected message %q", msg)
	}
}

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
