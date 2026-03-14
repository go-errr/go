package err

type IllegalStateException struct {
	*RuntimeException
}

func NewIllegalStateException(message string) *IllegalStateException {
	return &IllegalStateException{
		RuntimeException: NewRuntimeExceptionWith(message, nil, StackTrace(1)),
	}
}

func NewIllegalStateExceptionFrom(message string, cause any) *IllegalStateException {
	return &IllegalStateException{
		RuntimeException: NewRuntimeExceptionWith(message, cause, StackTrace(1)),
	}
}

func NewIllegalStateExceptionWith(message string, cause any, stackTrace []uintptr) *IllegalStateException {
	return &IllegalStateException{
		RuntimeException: NewRuntimeExceptionWith(message, cause, stackTrace),
	}
}
