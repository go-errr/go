package err

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
)

type UncaughtExceptionHandler func(any)

var defaultUncaughtExceptionHandler atomic.Value

/*
Recover intercepts an uncaught panic at an outer execution boundary.

It is intended primarily for goroutine entrypoints, background workers,
schedulers, callbacks, and other asynchronous execution roots where an
uncaught panic would otherwise terminate the whole process.

Recover must be used with defer.

If no handlers are provided, Recover delegates to the default uncaught
exception handler. If no default handler is configured, a stack trace is
printed to stderr.

Recover is a containment mechanism, not a business error handling tool.
Local error classification and propagation policy should normally be
implemented using err.Catch.

Typical usage:

	go func() {
		defer err.Recover(func(e any) {
			slog.Error("Backup job crashed " + err.PrintStackTrace(e))
			updateJobStatus()
		})

		runBackupJob()
	}()
*/
func Recover(handler ...func(any)) {
	if e := recover(); e != nil {
		if len(handler) == 0 {
			uncaughtExceptionHandler := DefaultUncaughtExceptionHandler()
			if uncaughtExceptionHandler != nil {
				uncaughtExceptionHandler(e)
			} else {
				fmt.Fprintf(os.Stderr, "panic: %s\n", PrintStackTrace(e))
			}
			return
		}
		for _, handler := range handler {
			doHandle(e, handler)
		}
	}
}

func doHandle(e any, handler func(any)) {
	defer Recover()
	handler(e)
}

/*
Catch intercepts a panic, passes the recovered value to the provided handler,
and stops stack unwinding unless the handler explicitly re-panics.

Catch must be used with defer and is intended to be the primary panic handling
mechanism in application code. It provides semantics similar to a Java
`catch (Throwable e)` block.

The handler receives the original panic value (which may or may not implement
error) and is fully responsible for deciding the propagation policy:

  - swallow the panic and continue execution
  - perform fallback logic
  - wrap the panic into a domain or runtime exception
  - log or emit metrics
  - rethrow using panic(e) to continue unwinding

Catch never re-panics automatically.

Typical usage:

	defer err.Catch(func(e any) {
		switch {
		case err.Is(e, context.Canceled, io.EOF):
			slog.Info("operation stopped", "err", e)

		case err.As2[*os.PathError, *net.OpError](e):
			slog.Warn("external resource unavailable", "err", e)
			scheduleRetry()

		default:
			panic(e)
		}
	})
*/
func Catch(handler ...func(any)) {
	if e := recover(); e != nil {
		for _, handler := range handler {
			handler(e)
		}
	}
}

/*
Assert panics with AssertionError when condition is false.

Use it for internal invariants and programmer assumptions.

	err.Assert(port > 0, "invalid port: %d", port)
*/
func Assert(condition bool, format string, args ...any) {
	if !condition {
		panic(NewAssertionError(fmt.Sprintf(format, args...)))
	}
}

/*
Is is a convenience wrapper over errors.Is that accepts any and multiple targets.

It returns true if err matches at least one target.

	if err.Is(e, fs.ErrExist, os.ErrExist) {
		return
	}
*/
func Is(err any, targets ...error) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	for _, t := range targets {
		if errors.Is(e, t) {
			return true
		}
	}
	return false
}

/*
As is a generic wrapper over errors.As.

It returns the matched typed error and true on success.

	if pe, ok := err.As[*fs.PathError](e); ok {
		slog.Warn("path error " + pe.Path)
	}
*/
func As[T error](err any) (T, bool) {
	var zero T
	e, ok := err.(error)
	if !ok {
		return zero, false
	}

	var target T
	if errors.As(e, &target) {
		return target, true
	}

	return zero, false
}

/*
As1 reports whether err matches T1 using errors.As.

Use it in switch cases when only the match result matters.

	if err.As1[*fs.PathError](e) {
		return
	}
*/
func As1[T1 error](err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	var t1 T1
	return errors.As(e, &t1)
}

/*
As2 reports whether err matches T1 or T2 using errors.As.

Use it for Java-like multi-catch checks.

	if err.As2[*err.IllegalStateException, *err.IllegalArgumentException](e) {
		return
	}
*/
func As2[T1 error, T2 error](err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	var t1 T1
	if errors.As(e, &t1) {
		return true
	}
	var t2 T2
	return errors.As(e, &t2)
}

/*
As3 reports whether err matches T1, T2, or T3 using errors.As.
*/
func As3[T1 error, T2 error, T3 error](err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	var t1 T1
	if errors.As(e, &t1) {
		return true
	}
	var t2 T2
	if errors.As(e, &t2) {
		return true
	}
	var t3 T3
	return errors.As(e, &t3)
}

/*
As4 reports whether err matches T1, T2, T3, or T4 using errors.As.
*/
func As4[T1 error, T2 error, T3 error, T4 error](err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	var t1 T1
	if errors.As(e, &t1) {
		return true
	}
	var t2 T2
	if errors.As(e, &t2) {
		return true
	}
	var t3 T3
	if errors.As(e, &t3) {
		return true
	}
	var t4 T4
	return errors.As(e, &t4)
}

/*
As5 reports whether err matches T1, T2, T3, T4, or T5 using errors.As.
*/
func As5[T1 error, T2 error, T3 error, T4 error, T5 error](err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	var t1 T1
	if errors.As(e, &t1) {
		return true
	}
	var t2 T2
	if errors.As(e, &t2) {
		return true
	}
	var t3 T3
	if errors.As(e, &t3) {
		return true
	}
	var t4 T4
	if errors.As(e, &t4) {
		return true
	}
	var t5 T5
	return errors.As(e, &t5)
}

/*
AsAny checks whether err matches at least one target type using errors.As.

Use it when the number of target types is dynamic or greater than As5 supports.

	if err.AsAny(e,
		err.Type[*err.IllegalStateException](),
		err.Type[*err.IllegalArgumentException]()) {
		return
	}
*/
func AsAny(err any, targets ...any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	for _, target := range targets {
		if errors.As(e, target) {
			return true
		}
	}
	return false
}

/*
Type creates a typed zero target for AsAny.

It is a helper for concise type-based matching.

	err.Type[*fs.PathError]()
	err.Type[*err.RuntimeException]()
*/
func Type[T error]() *T {
	var zero T
	return &zero
}

/*
Interrupted reports whether err represents a cooperative interruption signal.

It returns true for InterruptedException and for context.Canceled.

	defer err.Recover(func(e any) {
		if err.Interrupted(e) {
			return // silent cooperative stop
		}
		slog.Error("operation failed " + err.PrintStackTrace(e))
	})
*/
func Interrupted(err any) bool {
	if err == nil {
		return false
	}
	e, ok := err.(error)
	if !ok {
		return false
	}

	var interrupted *InterruptedException
	return errors.As(e, &interrupted) ||
		errors.Is(e, context.Canceled)
}

func StackTrace(skipFrames ...int) []uintptr {
	skip := 2
	if len(skipFrames) > 0 {
		skip += skipFrames[0]
	}
	pcs := make([]uintptr, 64)
	for {
		n := runtime.Callers(skip, pcs)
		if n < len(pcs) {
			return append([]uintptr(nil), pcs[:n]...)
		}
		pcs = make([]uintptr, len(pcs)*2)
	}
}

func PrintStackTrace(e any) string {
	if e == nil {
		return ""
	}
	e1, ok := e.(error)
	if !ok {
		return fmt.Sprint(e)
	}
	var b strings.Builder
	for i := 0; e1 != nil; i++ {
		if i == 0 {
			fmt.Fprintf(&b, "%T: %v\n", e1, e1)
		} else {
			fmt.Fprintf(&b, "Caused by: %T: %v\n", e1, e1)
		}
		if st, ok := e1.(interface{ StackTrace() []uintptr }); ok {
			stack := formatStackTrace(st.StackTrace())
			if stack != "" {
				b.WriteString(stack)
				b.WriteByte('\n')
			}
		}
		e1 = errors.Unwrap(e1)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatStackTrace(pcs []uintptr) string {
	if len(pcs) == 0 {
		return ""
	}
	var b strings.Builder
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		if frame.Function != "runtime.goexit" && frame.Function != "runtime.main" {
			fmt.Fprintf(&b, "\tat %s (%s:%d)\n", frame.Function, frame.File, frame.Line)
		}
		if !more {
			break
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func DefaultUncaughtExceptionHandler() UncaughtExceptionHandler {
	handler, _ := defaultUncaughtExceptionHandler.Load().(UncaughtExceptionHandler)
	return handler
}

func SetDefaultUncaughtExceptionHandler(handler UncaughtExceptionHandler) {
	defaultUncaughtExceptionHandler.Store(handler)
}
