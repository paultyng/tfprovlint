package provparse

import (
	"bytes"
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ssa"
)

func funcInstructions(f *ssa.Function) []ssa.Instruction {
	if f.Blocks == nil {
		return nil
	}

	instrs := []ssa.Instruction{}
	for _, b := range f.Blocks {
		instrs = append(instrs, b.Instrs...)
	}
	return instrs
}

// inspectInstructions walks instructions follows calls
func inspectInstructions(instrs []ssa.Instruction, cb func(ins ssa.Instruction) bool) {
	visited := map[*ssa.Function]bool{}

	var walk func(instrs []ssa.Instruction) bool
	walk = func(instrs []ssa.Instruction) bool {
		for _, ins := range instrs {
			if !cb(ins) {
				return false
			}

			ssacall, ok := ins.(ssa.CallInstruction)
			if !ok {
				continue
			}

			if callee := ssacall.Common().StaticCallee(); callee != nil {
				if visited[callee] {
					return true
				}
				visited[callee] = true
				calleeInstrs := funcInstructions(callee)
				if !walk(calleeInstrs) {
					return false
				}
			}

		}

		return true
	}
	walk(instrs)
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
		case ssa.CallInstruction:
			if callee := v.Common().StaticCallee(); callee != nil {
				if visited[callee] {
					return v.(ssa.Value)
				}
				visited[callee] = true
				calleeInstrs := funcInstructions(callee)
				var retValue ssa.Value
				inspectInstructions(calleeInstrs, func(ins ssa.Instruction) bool {
					if ret, ok := ins.(*ssa.Return); ok {
						// assume first result?
						retValue = ret.Results[0]
						return false
					}
					return true
				})
				return retValue
			}
		}
		return v
	}
	return walk(v)
}

func structFieldStringValue(instrs []ssa.Instruction, structType, fieldName string) string {
	v := structFieldValue(instrs, structType, fieldName)
	if v == nil {
		return ""
	}
	v = rootValue(v)

	switch v := v.(type) {
	case *ssa.Const:
		s, err := strconv.Unquote(v.Value.ExactString())
		if err != nil {
			panic(fmt.Sprintf("unable to unquote string value: %s", err))
		}
		return s
	default:
		panic(fmt.Sprintf("unexpected value type %T", v))
	}
}

func structFieldBoolValue(instrs []ssa.Instruction, structType, fieldName string) bool {
	v := structFieldValue(instrs, structType, fieldName)
	if v == nil {
		return false
	}
	v = rootValue(v)

	switch v := v.(type) {
	case *ssa.Const:
		b, err := strconv.ParseBool(v.Value.ExactString())
		if err != nil {
			panic(fmt.Sprintf("unable to parse bool: %s", err))
		}
		return b
	default:
		panic(fmt.Sprintf("unexpected value type %T", v))
	}
}

func structFieldValue(instrs []ssa.Instruction, structType, fieldName string) ssa.Value {
	var store *ssa.Store
	inspectInstructions(instrs, func(ins ssa.Instruction) bool {
		fieldAddr, ok := ins.(*ssa.FieldAddr)
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
		inspectInstructions(*fieldAddr.Referrers(), func(ins ssa.Instruction) bool {
			var ok bool
			if store, ok = ins.(*ssa.Store); ok {
				return false
			}
			return true
		})
		if store != nil {
			return false
		}
		return true
	})
	if store == nil {
		return nil
	}

	return store.Val
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
