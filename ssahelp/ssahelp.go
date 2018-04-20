package ssahelp

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type ErrNoExpectedValueFound struct {
	Found ssa.Value
}

func (err *ErrNoExpectedValueFound) Error() string {
	return fmt.Sprintf("expected value not found, found %T", err.Found)
}

func IsNoExpectedValueFound(err error) bool {
	_, ok := err.(*ErrNoExpectedValueFound)
	return ok
}

type ErrNoFieldAddrFound struct{}

func (err *ErrNoFieldAddrFound) Error() string {
	return "no field assignment found"
}

func IsNoFieldAddrFound(err error) bool {
	_, ok := err.(*ErrNoFieldAddrFound)
	return ok
}

func DerefType(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		return DerefType(ptr.Elem())
	}
	return t
}

func FuncInstructions(f *ssa.Function) []ssa.Instruction {
	if f.Blocks == nil {
		return nil
	}

	instrs := []ssa.Instruction{}
	for _, b := range f.Blocks {
		instrs = append(instrs, b.Instrs...)
	}
	return instrs
}

// InspectInstructions walks instructions follows calls
func InspectInstructions(instrs []ssa.Instruction, cb func(ins ssa.Instruction) bool) {
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
					continue
				}
				visited[callee] = true
				calleeInstrs := FuncInstructions(callee)
				if !walk(calleeInstrs) {
					return false
				}
			}

		}

		return true
	}
	walk(instrs)
}

func ReturnValue(f *ssa.Function, index int) ssa.Value {
	if f.Signature.Results().Len() == 0 {
		return nil
	}

	instrs := FuncInstructions(f)
	// reverse the instructions to find last return
	for i := len(instrs) - 1; i >= 0; i-- {
		// only walk to the first return encountered for now
		if ret, ok := instrs[i].(*ssa.Return); ok {
			return ret.Results[index]
		}
	}
	return nil
}

func RootValue(v ssa.Value) ssa.Value {
	path := RootValuePath(v)
	if len(path) == 0 {
		return nil
	}
	return path[len(path)-1]
}

func RootValuePath(v ssa.Value) []ssa.Value {
	var path []ssa.Value

	visited := func(v ssa.Value) bool {
		for _, p := range path {
			if p == v {
				return true
			}
		}
		return false
	}

	var walk func(ssa.Value, int)
	walk = func(v ssa.Value, index int) {
		if visited(v) {
			// error? infinite loop at least
			// TODO: trace this
			return
		}

		path = append(path, v)
		switch v := v.(type) {
		case *ssa.UnOp:
			switch v.Op {
			case token.ARROW:
				// reset index?
				walk(v.X, 0)
				return
			case token.MUL:
				walk(v.X, index)
				return
			default:
				return
			}
		case *ssa.Extract:
			// need to reset this index after any Call, TypeAssert, Next, UnOp(ARROW) and IndexExpr(Map)
			walk(v.Tuple, v.Index)
			return
		case *ssa.MakeClosure:
			walk(v.Fn, index)
			return
		case *ssa.MakeInterface:
			walk(v.X, index)
			return
		case *ssa.ChangeType:
			walk(v.X, index)
			return
		case ssa.CallInstruction:
			if callee := v.Common().StaticCallee(); callee != nil {
				if visited(callee) {
					// error? infinite loop at least
					// TODO: trace this
					return
				}
				path = append(path, callee)
				retValue := ReturnValue(callee, index)
				// consume the index from extract
				walk(retValue, 0)
				return
			}
			//TODO: trace this is a unrecognized call type
		}
	}
	walk(v, 0)
	return path
}

func StructFieldStringValue(instrs []ssa.Instruction, structType, fieldName string) (string, error) {
	v, err := StructFieldValue(instrs, structType, fieldName)
	if err != nil {
		return "", err
	}
	v = RootValue(v)

	switch v := v.(type) {
	case *ssa.Const:
		s, err := strconv.Unquote(v.Value.ExactString())
		if err != nil {
			return "", err
		}
		return s, nil
	default:
		return "", &ErrNoExpectedValueFound{
			Found: v,
		}
	}
}

func StructFieldBoolValue(instrs []ssa.Instruction, structType, fieldName string) (bool, error) {
	v, err := StructFieldValue(instrs, structType, fieldName)
	if err != nil {
		return false, err
	}
	v = RootValue(v)
	test := true
	if unop, ok := v.(*ssa.UnOp); ok && unop.Op == token.NOT {
		test = false
		v = RootValue(unop.X)
	}

	switch v := v.(type) {
	case *ssa.Const:
		b, err := strconv.ParseBool(v.Value.ExactString())
		if err != nil {
			return false, err
		}
		return b && test, nil
	default:
		return false, &ErrNoExpectedValueFound{
			Found: v,
		}
	}
}

func StructFieldFuncValue(instrs []ssa.Instruction, structType, fieldName string) (*ssa.Function, error) {
	v, err := StructFieldValue(instrs, structType, fieldName)
	if err != nil {
		return nil, err
	}
	v = RootValue(v)
	switch v := v.(type) {
	case *ssa.Function:
		return v, nil
	}
	return nil, &ErrNoExpectedValueFound{
		Found: v,
	}
}

func FieldAddrValue(fieldAddr *ssa.FieldAddr) ssa.Value {
	var store *ssa.Store
	InspectInstructions(*fieldAddr.Referrers(), func(ins ssa.Instruction) bool {
		var ok bool
		if store, ok = ins.(*ssa.Store); ok {
			return false
		}
		return true
	})
	return store.Val
}

func FieldAddrField(fieldAddr *ssa.FieldAddr) *types.Var {
	t := fieldAddr.X.Type()
	t = DerefType(t)
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	strc, ok := t.(*types.Struct)
	if !ok {
		return nil
	}
	field := strc.Field(fieldAddr.Field)
	return field
}

func StructFieldValue(instrs []ssa.Instruction, structType, fieldName string) (ssa.Value, error) {
	var v ssa.Value
	InspectInstructions(instrs, func(ins ssa.Instruction) bool {
		fieldAddr, ok := ins.(*ssa.FieldAddr)
		if !ok {
			return true
		}
		t := fieldAddr.X.Type()
		t = DerefType(t)
		if !TypeMatch(t, structType) {
			return true
		}
		field := FieldAddrField(fieldAddr)
		if field == nil || field.Name() != fieldName {
			return true
		}
		v = FieldAddrValue(fieldAddr)
		return v != nil
	})
	if v == nil {
		return nil, &ErrNoFieldAddrFound{}
	}

	return v, nil
}

func TypeMatch(t types.Type, name string) bool {
	buf := &bytes.Buffer{}
	types.WriteType(buf, t, NormalizePkgPath)
	normalizedTypeName := buf.String()
	return name == normalizedTypeName
}

func NormalizePkgPath(pkg *types.Package) string {
	const vendor = "/vendor/"
	pkgPath := pkg.Path()
	if i := strings.LastIndex(pkgPath, vendor); i != -1 {
		return pkgPath[i+len(vendor):]
	}
	return pkgPath
}
