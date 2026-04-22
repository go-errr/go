package err

import "fmt"

type NumberFormatException struct {
	IllegalArgumentException
}

func NewNumberFormatException(message string) *NumberFormatException {
	return &NumberFormatException{
		IllegalArgumentException: *NewIllegalArgumentExceptionWith(message, nil, StackTrace(1)),
	}
}

func NewNumberFormatExceptionFrom(message string, cause any) *NumberFormatException {
	return &NumberFormatException{
		IllegalArgumentException: *NewIllegalArgumentExceptionWith(message, cause, StackTrace(1)),
	}
}

func NewNumberFormatExceptionWith(message string, cause any, stackTrace []uintptr) *NumberFormatException {
	return &NumberFormatException{
		IllegalArgumentException: *NewIllegalArgumentExceptionWith(message, cause, stackTrace),
	}
}

func (this *NumberFormatException) Format(s fmt.State, verb rune) {
	this.DefaultFormat(s, verb, this)
}
