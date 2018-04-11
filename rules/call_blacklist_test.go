package rules

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestFunctionCalls(t *testing.T) {
	pkg, err := makeSamplePkg(`
package test

import (
	"bytes"
	"fmt"
)

func foo() {
	bar()
	fmt.Println("")
}

func bar() {
	baz()
	bar()
}

func baz() {
	b := bytes.NewBuffer(([]byte)(""))
	b.Len()
}		
`)
	if err != nil {
		t.Fatalf("unable to build test package: %s", err)
	}

	b := bytes.NewBuffer(([]byte)(""))
	b.Len()

	for i, c := range []struct {
		expected []string
		funcName string
		methods  []string
	}{
		// works with empty list
		{[]string{}, "foo", []string{}},
		{[]string{}, "bar", []string{}},
		{[]string{}, "baz", []string{}},

		// no false positives
		{[]string{}, "foo", []string{"notfound"}},
		{[]string{}, "bar", []string{"notfound"}},
		{[]string{}, "baz", []string{"notfound"}},

		// finds, even nested calls
		{[]string{"test.baz"}, "foo", []string{"test.baz", "notfound"}},
		{[]string{"test.baz"}, "bar", []string{"test.baz", "notfound"}},

		// shouldn't find itself
		{[]string{}, "baz", []string{"test.baz", "notfound"}},
		// finds recursion if self
		{[]string{"test.bar"}, "bar", []string{"test.bar"}},

		//finds multiple
		{[]string{"test.bar", "test.baz"}, "foo", []string{"test.bar", "test.baz", "notfound"}},

		//finds other packages' static methods
		{[]string{"fmt.Println"}, "foo", []string{"fmt.Println"}},

		//finds instance methods
		{[]string{"(*bytes.Buffer).Len"}, "foo", []string{"(*bytes.Buffer).Len"}},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actualPos := functionCalls(pkg.Func(c.funcName), stringSliceToSet(c.methods))

			actual := make([]string, 0, len(actualPos))
			for k := range actualPos {
				actual = append(actual, k)
			}

			if !reflect.DeepEqual(c.expected, actual) {
				t.Fatalf("expected %q does not match %q", c.expected, actual)
			}
		})
	}

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
	pkg.Build()
	if err != nil {
		return nil, err
	}
	return pkg, nil
}
