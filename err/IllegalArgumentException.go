package err

import "fmt"

type IllegalArgumentException struct {
	*RuntimeException
}

func NewIllegalArgumentException(message string) *IllegalArgumentException {
	return &IllegalArgumentException{
		RuntimeException: NewRuntimeExceptionWith(message, nil, StackTrace(1)),
	}
}

func NewIllegalArgumentExceptionFrom(message string, cause any) *IllegalArgumentException {
	return &IllegalArgumentException{
		RuntimeException: NewRuntimeExceptionWith(message, cause, StackTrace(1)),
	}
}

func NewIllegalArgumentExceptionWith(message string, cause any, stackTrace []uintptr) *IllegalArgumentException {
	return &IllegalArgumentException{
		RuntimeException: NewRuntimeExceptionWith(message, cause, stackTrace),
	}
}

func (this *IllegalArgumentException) Format(s fmt.State, verb rune) {
	this.DefaultFormat(s, verb, this)
}
