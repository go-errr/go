package err

import (
	"fmt"
	"io"
)

/*
The Throwable is the superclass of all errors and exceptions.

A throwable contains a snapshot of the execution stack of its thread at the time it was created.
*/
type AbstractThrowable struct {
	message string
	cause   error
	stack   []uintptr
}

func NewAbstractThrowable(message string, cause any, stackTrace []uintptr) *AbstractThrowable {
	var err error
	switch v := cause.(type) {
	case nil:
		err = nil
	case error:
		err = v
	default:
		err = fmt.Errorf("%v", v)
	}
	if stackTrace == nil {
		stackTrace = StackTrace(2)
	}

	return &AbstractThrowable{
		message: message,
		cause:   err,
		stack:   stackTrace,
	}
}

func (this *AbstractThrowable) Error() string {
	return this.message
}

func (this *AbstractThrowable) Unwrap() error {
	return this.cause
}

func (this *AbstractThrowable) StackTrace() []uintptr {
	return this.stack
}

/*
Default implementation of fmt.Formatter.

Supported verbs:

	%s  message
	%v  message
	%+v stack trace
	%q  quoted message
*/
func (this *AbstractThrowable) DefaultFormat(s fmt.State, verb rune, e error) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, PrintStackTrace(e))
		} else {
			io.WriteString(s, e.Error())
		}
	case 's':
		io.WriteString(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	default:
		io.WriteString(s, e.Error())
	}
}
