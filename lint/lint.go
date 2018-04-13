package lint

import (
	"fmt"
	"go/token"

	"github.com/paultyng/tfprovlint/provparse"
)

// Issue represents a potential problem found in the code.
type Issue struct {
	Message string
	Pos     token.Pos
}

// NewIssuef is a helper to create an issue from a string format.
func NewIssuef(pos token.Pos, format string, args ...interface{}) Issue {
	iss := Issue{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
	}

	return iss
}

// ResourceRule is a rule that is evaluated against resource (and data source) data.
type ResourceRule interface {
	CheckResource(*provparse.Resource) ([]Issue, error)
}

// ProviderRule is a rule that is evaluated against provider data.
type ProviderRule interface {
	CheckProvider(*provparse.Provider) ([]Issue, error)
}
