package provparse

import (
	"fmt"
	"go/token"
)

type posError struct {
	pos   token.Pos
	inner error
	msg   string
}

var _ error = &posError{}

func (err *posError) Error() string {
	return err.msg
}

func (err *posError) cause() *posError {
	var innerMost *posError
	for posErr := err; posErr != nil; posErr, _ = posErr.inner.(*posError) {
		innerMost = posErr
	}
	return innerMost
}

func (err *posError) Position() token.Pos {
	return err.cause().pos
}

func (err *posError) Cause() error {
	c := err.cause()

	if c.inner != nil {
		return c.inner
	}

	return c
}

type poser interface {
	Pos() token.Pos
}

func wrapNodeErrorf(err error, pos poser, format string, args ...interface{}) error {
	return &posError{
		inner: err,
		pos:   pos.Pos(),
		msg:   fmt.Sprintf(format, args...),
	}
}

func nodeErrorf(pos poser, format string, args ...interface{}) error {
	return wrapNodeErrorf(nil, pos, format, args...)
}

func unwrapError(err error, fset *token.FileSet) error {
	switch err := err.(type) {
	case *posError:
		cause := err.Cause()
		if cause != err {
			return fmt.Errorf("%s: %s: %s", fset.Position(err.Position()), err.Error(), cause.Error())
		}
		return fmt.Errorf("%s: %s", fset.Position(err.Position()), err.Error())
	}

	return err
}
