package lang

import (
	"fmt"
	"reflect"

	"github.com/go-errr/go/err"
)

func If[T any](cond bool, v1, v2 T) T {
	if cond {
		return v1
	}
	return v2
}

func IsNil(value any) bool {
	if value == nil {
		return true
	}
	defer func() { recover() }()
	return reflect.ValueOf(value).IsNil()
}

func TypeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

/*
Assert verifies that the provided expression is true.

If expression is false, Assert panics with an AssertionError containing the
formatted message.

Example:

	lang.Assert(user != nil, "User must not be nil")
	lang.Assert(len(items) > 0, "Items collection must not be empty")
*/
func Assert(expression bool, format string, args ...any) {
	if !expression {
		panic(err.NewAssertionError(fmt.Sprintf(format, args...)))
	}
}
