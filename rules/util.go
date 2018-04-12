package rules

import (
	"bytes"
	"fmt"
	"go/types"
	"log"
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

func inspectValue(root ssa.Value, cb func(v ssa.Value) bool) {
	visited := map[ssa.Value]bool{}

	var walk func(v ssa.Value) bool
	walk = func(v ssa.Value) bool {
		if visited[v] {
			return true
		}
		visited[v] = true

		if !cb(v) {
			return false
		}

		switch v := v.(type) {
		case *ssa.MakeInterface:
			if !walk(v.X) {
				return false
			}
		//TODO: need to simulate this case and make sure I want it
		case *ssa.Phi:
			log.Printf("[WARN] um, I don't really understand Phi... here be dragons...")
			for _, edge := range v.Edges {
				if !walk(edge) {
					return false
				}
			}
		}

		return true
	}
	walk(root)
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
