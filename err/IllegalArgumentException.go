package err

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
