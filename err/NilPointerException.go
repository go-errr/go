package err

import "fmt"

type NilPointerException struct {
	RuntimeException
}

func NewNilPointerException(message string) *NilPointerException {
	return &NilPointerException{
		RuntimeException: *NewRuntimeExceptionWith(message, nil, StackTrace(1)),
	}
}

func NewNilPointerExceptionFrom(message string, cause any) *NilPointerException {
	return &NilPointerException{
		RuntimeException: *NewRuntimeExceptionWith(message, cause, StackTrace(1)),
	}
}

func NewNilPointerExceptionWith(message string, cause any, stackTrace []uintptr) *NilPointerException {
	return &NilPointerException{
		RuntimeException: *NewRuntimeExceptionWith(message, cause, stackTrace),
	}
}

func (this *NilPointerException) Format(s fmt.State, verb rune) {
	this.DefaultFormat(s, verb, this)
}
