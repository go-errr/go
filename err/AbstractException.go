package err

/*
The class Exception and its subclasses are a form of Throwable that indicates conditions that a reasonable application might want to catch.
*/
type AbstractException struct {
	*AbstractThrowable
}

func NewAbstractException(message string) *AbstractException {
	return &AbstractException{
		AbstractThrowable: NewAbstractThrowable(message, nil, nil),
	}
}
func NewAbstractExceptionFrom(message string, cause any) *AbstractException {
	return &AbstractException{
		AbstractThrowable: NewAbstractThrowable(message, cause, nil),
	}
}
func NewAbstractExceptionWith(message string, cause any, stackTrace []uintptr) *AbstractException {
	return &AbstractException{
		AbstractThrowable: NewAbstractThrowable(message, cause, stackTrace),
	}
}
