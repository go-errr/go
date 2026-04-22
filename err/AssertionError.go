package err

type AssertionError struct {
	AbstractError
}

func NewAssertionError(message string) *AssertionError {
	return &AssertionError{
		AbstractError: *NewAbstractError(message, nil, StackTrace(1)),
	}
}

func NewAssertionErrorFrom(message string, cause any) *AssertionError {
	return &AssertionError{
		AbstractError: *NewAbstractError(message, cause, StackTrace(1)),
	}
}

func NewAssertionErrorWith(message string, cause any, stackTrace []uintptr) *AssertionError {
	return &AssertionError{
		AbstractError: *NewAbstractError(message, cause, stackTrace),
	}
}
