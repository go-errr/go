package err

type RuntimeException struct {
	*AbstractException
}

func NewRuntimeException(message string) *RuntimeException {
	return &RuntimeException{
		AbstractException: NewAbstractException(message),
	}
}

func NewRuntimeExceptionFrom(message string, cause any) *RuntimeException {
	return &RuntimeException{
		AbstractException: NewAbstractExceptionFrom(message, cause),
	}
}

func NewRuntimeExceptionWith(message string, cause any, stackTrace []uintptr) *RuntimeException {
	return &RuntimeException{
		AbstractException: NewAbstractExceptionWith(message, cause, stackTrace),
	}
}
