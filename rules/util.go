package rules

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"
)

func normalizePkgPath(pkg *types.Package) string {
	const vendor = "/vendor/"
	pkgPath := pkg.Path()
	if i := strings.LastIndex(pkgPath, vendor); i != -1 {
		return pkgPath[i+len(vendor):]
	}
	return pkgPath
}

func normalizeSSAFunctionString(f *ssa.Function) string {
	funcName := f.Name()

	if recv := f.Signature.Recv(); recv != nil {
		//pkgPath := normalizePkgPath(recv.Pkg())
		buf := &bytes.Buffer{}
		types.WriteType(buf, recv.Type(), normalizePkgPath)
		typeName := buf.String()

		return fmt.Sprintf("(%s).%s", typeName, funcName)
	}

	pkgPath := normalizePkgPath(f.Pkg.Pkg)

	return fmt.Sprintf("%s.%s", pkgPath, funcName)
}

func valueBeforeInterface(v ssa.Value) ssa.Value {
	if v, ok := v.(*ssa.MakeInterface); ok {
		return v.X
	}

	return v
}

func inspectInstructions(root *ssa.Function, cb func(ins ssa.Instruction) bool) {
	visited := map[*ssa.Function]bool{}

	var walk func(f *ssa.Function) bool
	walk = func(f *ssa.Function) bool {
		if visited[f] {
			// log.Printf("[TRACE] already visited function %s", f.String())
			return true
		}
		visited[f] = true

		if f.Blocks == nil {
			// log.Printf("[TRACE] ignoring external function %s", f.String())
			return true
		}

		for _, b := range f.Blocks {
			for _, ins := range b.Instrs {
				if !cb(ins) {
					return false
				}

				ssacall, ok := ins.(ssa.CallInstruction)
				if !ok {
					continue
				}

				if callee := ssacall.Common().StaticCallee(); callee != nil {
					if !walk(callee) {
						return false
					}
				}

			}
		}

		return true
	}
	walk(root)
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
