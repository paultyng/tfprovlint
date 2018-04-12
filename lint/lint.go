package lint

import (
	"fmt"
	"go/token"

	"github.com/paultyng/tfprovlint/provparse"
)

type Issue struct {
	Message string
	Pos     token.Pos
}

func NewIssuef(pos token.Pos, format string, args ...interface{}) Issue {
	iss := Issue{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
	}

	return iss
}

type ResourceRule interface {
	CheckResource(*provparse.Resource) ([]Issue, error)
}
