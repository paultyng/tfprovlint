package rules

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"log"
	"strings"

	"github.com/paultyng/tfprovlint/lint"
	"github.com/paultyng/tfprovlint/provparse"
	"golang.org/x/tools/go/ssa"
)

const ruleIDDeleteShouldNotCallSetId = "tfprovlint001"

var (
	defaultDeleteBlacklist = []string{
		"(*github.com/hashicorp/terraform/helper/schema.ResourceData).SetId",
	}
)

type callBlacklist struct {
	Delete map[string]bool
}

var _ lint.ResourceRule = &callBlacklist{}

func stringSliceToSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

func NewCallBlacklist() lint.ResourceRule {
	return &callBlacklist{
		Delete: stringSliceToSet(defaultDeleteBlacklist),
	}
}

func (rule *callBlacklist) CheckResource(r *provparse.Resource) ([]lint.Issue, error) {
	var issues []lint.Issue

	if r.DeleteFunc != nil {
		if calls := functionCalls(r.DeleteFunc, rule.Delete); len(calls) > 0 {
			// it makes some of the calls, need to append issues
			for call, pos := range calls {
				issues = append(issues, lint.NewIssuef(ruleIDDeleteShouldNotCallSetId, pos, "DeleteFunc should not call %s", call))
			}
		}
	}

	return issues, nil
}

func functionCalls(f *ssa.Function, callList map[string]bool) map[string]token.Pos {
	calls := map[string]token.Pos{}
	visited := map[*ssa.Function]bool{}

	var walk func(f *ssa.Function)
	walk = func(f *ssa.Function) {
		if visited[f] {
			log.Printf("[DEBUG] already visited function %s", f.String())
			return
		}
		visited[f] = true

		if f.Blocks == nil {
			log.Printf("[DEBUG] ignoring external function %s", f.String())
			return
		}

		for _, b := range f.Blocks {
			for _, ins := range b.Instrs {
				ssacall, ok := ins.(ssa.CallInstruction)
				if !ok {
					continue
				}

				if callee := ssacall.Common().StaticCallee(); callee != nil {
					calleeName := normalizeSSAFunctionString(callee)
					log.Printf("[DEBUG] checking %q against list", calleeName)
					if callList[calleeName] {
						calls[calleeName] = ssacall.Pos()
					}

					walk(callee)
				}

			}
		}
	}
	walk(f)

	return calls
}

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
