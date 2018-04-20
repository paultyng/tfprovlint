package rules

import (
	"bytes"
	"fmt"
	"go/types"
	"log"
	"os"

	"golang.org/x/tools/go/ssa"

	"github.com/paultyng/tfprovlint/ssahelp"
)

type commonRule struct {
}

var (
	shouldTrace = false
	shouldWarn  = false
)

// TODO: do this better!!
func init() {
	switch os.Getenv("LOG_LVL") {
	case "TRACE":
		shouldTrace = true
		fallthrough
	case "WARN":
		shouldWarn = true
	}
}

func (*commonRule) tracef(format string, args ...interface{}) {
	if shouldTrace {
		log.Printf("[TRACE] "+format, args...)
	}
}

func (*commonRule) warnf(format string, args ...interface{}) {
	if shouldWarn {
		log.Printf("[WARN] "+format, args...)
	}
}

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
