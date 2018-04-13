package provparse

import (
	"bytes"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// inspectInstructions walks instructions in a function and follows calls
func inspectInstructions(start *ssa.Function, cb func(ins ssa.Instruction) bool) {
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
	walk(start)
}

func rootValue(v ssa.Value) ssa.Value {
	visited := map[ssa.Value]bool{}
	var walk func(v ssa.Value) ssa.Value
	walk = func(v ssa.Value) ssa.Value {
		if visited[v] {
			// error?
			return v
		}
		visited[v] = true

		switch v := v.(type) {
		case *ssa.MakeInterface:
			return walk(v.X)
		case *ssa.ChangeType:
			return walk(v.X)
		}
		return v
	}
	return walk(v)
}

func findStructFieldStore(f *ssa.Function, structType string, fieldName string) *ssa.Store {
	var store *ssa.Store
	inspectInstructions(f, func(ins ssa.Instruction) bool {
		var ok bool
		store, ok = ins.(*ssa.Store)
		if !ok {
			return true
		}
		fieldAddr, ok := store.Addr.(*ssa.FieldAddr)
		if !ok {
			return true
		}
		t := fieldAddr.X.Type()
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		if !typeMatch(t, structType) {
			return true
		}
		if named, ok := t.(*types.Named); ok {
			t = named.Underlying()
		}
		strc, ok := t.(*types.Struct)
		if !ok {
			return true
		}
		field := strc.Field(fieldAddr.Field)
		if field.Name() != fieldName {
			return true
		}
		return false
	})
	return store
}

func typeMatch(t types.Type, name string) bool {
	buf := &bytes.Buffer{}
	types.WriteType(buf, t, normalizePkgPath)
	normalizedTypeName := buf.String()
	return name == normalizedTypeName
}

func normalizePkgPath(pkg *types.Package) string {
	const vendor = "/vendor/"
	pkgPath := pkg.Path()
	if i := strings.LastIndex(pkgPath, vendor); i != -1 {
		return pkgPath[i+len(vendor):]
	}
	return pkgPath
}
