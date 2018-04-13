package rules

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/paultyng/tfprovlint/ssahelp"
	"golang.org/x/tools/go/ssa"
)

func normalizeSSAFunctionString(f *ssa.Function) string {
	funcName := f.Name()

	if recv := f.Signature.Recv(); recv != nil {
		//pkgPath := normalizePkgPath(recv.Pkg())
		buf := &bytes.Buffer{}
		types.WriteType(buf, recv.Type(), ssahelp.NormalizePkgPath)
		typeName := buf.String()

		return fmt.Sprintf("(%s).%s", typeName, funcName)
	}

	pkgPath := ssahelp.NormalizePkgPath(f.Pkg.Pkg)

	return fmt.Sprintf("%s.%s", pkgPath, funcName)
}

func numStars(v types.Type) int {
	stars := 0
	ptr, ok := v.(*types.Pointer)
	for ok {
		stars++
		ptr, ok = ptr.Elem().(*types.Pointer)
	}
	return stars
}
