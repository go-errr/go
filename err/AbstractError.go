package err

/*
The class Error and its subclasses are a form of Throwable that indicates serious problems that a reasonable application should not try to catch.
*/
type AbstractError struct {
	*AbstractThrowable
}

func NewAbstractError(message string) *AbstractError {
	return &AbstractError{
		AbstractThrowable: NewAbstractThrowable(message, nil, nil),
	}
}

func NewAbstractErrorFrom(message string, cause any) *AbstractError {
	return &AbstractError{
		AbstractThrowable: NewAbstractThrowable(message, cause, nil),
	}
}

func NewAbstractErrorWith(message string, cause any, stackTrace []uintptr) *AbstractError {
	return &AbstractError{
		AbstractThrowable: NewAbstractThrowable(message, cause, stackTrace),
	}
}
