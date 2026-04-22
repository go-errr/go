package err

import "fmt"

type RuntimeException struct {
	AbstractException
}

func NewRuntimeException(message string) *RuntimeException {
	return &RuntimeException{
		AbstractException: *NewAbstractException(message, nil, StackTrace(1)),
	}
}

func NewRuntimeExceptionFrom(message string, cause any) *RuntimeException {
	return &RuntimeException{
		AbstractException: *NewAbstractException(message, cause, StackTrace(1)),
	}
}

func NewRuntimeExceptionWith(message string, cause any, stackTrace []uintptr) *RuntimeException {
	return &RuntimeException{
		AbstractException: *NewAbstractException(message, cause, stackTrace),
	}
}

func (this *RuntimeException) Format(s fmt.State, verb rune) {
	this.DefaultFormat(s, verb, this)
}
