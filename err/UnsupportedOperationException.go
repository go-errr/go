package err

import "fmt"

type UnsupportedOperationException struct {
	RuntimeException
}

func NewUnsupportedOperationException(message string) *UnsupportedOperationException {
	return &UnsupportedOperationException{
		RuntimeException: *NewRuntimeExceptionWith(message, nil, StackTrace(1)),
	}
}

func NewUnsupportedOperationExceptionFrom(message string, cause any) *UnsupportedOperationException {
	return &UnsupportedOperationException{
		RuntimeException: *NewRuntimeExceptionWith(message, cause, StackTrace(1)),
	}
}

func NewUnsupportedOperationExceptionWith(message string, cause any, stackTrace []uintptr) *UnsupportedOperationException {
	return &UnsupportedOperationException{
		RuntimeException: *NewRuntimeExceptionWith(message, cause, stackTrace),
	}
}

func (this *UnsupportedOperationException) Format(s fmt.State, verb rune) {
	this.DefaultFormat(s, verb, this)
}
