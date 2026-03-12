package err

/*
InterruptedException represents a cooperative interruption signal.

It indicates that the current operation should stop as soon as possible and
propagate the interruption to the caller.

This exception is not treated as a failure and is typically excluded from
error logging and alerting pipelines.

Callers should use err.Interrupted to detect interruption conditions across
wrapped error chains.
*/
type InterruptedException struct {
	*AbstractException
}

func NewInterruptedException(message string) *InterruptedException {
	return &InterruptedException{
		AbstractException: NewAbstractException(message),
	}
}

func NewInterruptedExceptionFrom(message string, cause any) *InterruptedException {
	return &InterruptedException{
		AbstractException: NewAbstractExceptionFrom(message, cause),
	}
}

func NewInterruptedExceptionWith(message string, cause any, stackTrace []uintptr) *InterruptedException {
	return &InterruptedException{
		AbstractException: NewAbstractExceptionWith(message, cause, stackTrace),
	}
}
