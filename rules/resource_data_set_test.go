package rules

import (
	"fmt"

	"golang.org/x/tools/go/ssa"
)

func lookupSetCallInstruction(pkg *ssa.Package, funcName string) ssa.CallInstruction {
	f := pkg.Func(funcName)
	if f == nil {
		panic(fmt.Sprintf("func %q not found", funcName))
	}
	for _, b := range f.Blocks {
		for _, ins := range b.Instrs {
			ssacall, ok := ins.(ssa.CallInstruction)
			if !ok {
				continue
			}
			if ssacall.Common().StaticCallee().Name() == "Set" {
				return ssacall
			}
		}
	}
	return nil
}
