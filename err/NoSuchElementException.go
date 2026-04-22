package err

import "fmt"

type NoSuchElementException struct {
	RuntimeException
}

func NewNoSuchElementException(message string) *NoSuchElementException {
	return &NoSuchElementException{
		RuntimeException: *NewRuntimeExceptionWith(message, nil, StackTrace(1)),
	}
}

func NewNoSuchElementExceptionFrom(message string, cause any) *NoSuchElementException {
	return &NoSuchElementException{
		RuntimeException: *NewRuntimeExceptionWith(message, cause, StackTrace(1)),
	}
}

func NewNoSuchElementExceptionWith(message string, cause any, stackTrace []uintptr) *NoSuchElementException {
	return &NoSuchElementException{
		RuntimeException: *NewRuntimeExceptionWith(message, cause, stackTrace),
	}
}

func (this *NoSuchElementException) Format(s fmt.State, verb rune) {
	this.DefaultFormat(s, verb, this)
}
