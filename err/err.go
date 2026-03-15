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
Recover intercepts a panic, executes recovery logic, and always stops panic propagation.

Use it at execution boundaries where a panic must never escape, even if the recovery
logic itself fails. For example, at the beginning of a new goroutine.

The handler may perform logging, rollback, cleanup, or business state updates.
If the handler itself panics, stack unwinding is still stopped.

If no handler is provided, the default uncaught exception handler is used;
otherwise, the stack trace is printed to stderr.

Example:

	func runImportWorkerAsync(jobId string) {
		go func() {
			defer err.Recover(func(e any) {
				markJobFailed(jobId)      // may panic
				slog.Error("import worker failed " + err.PrintStackTrace(e))
			})

			processImport(jobId) // may panic
		}()
	}
*/
func Recover(handler ...func(any)) {
	if e := recover(); e != nil {
		if len(handler) == 0 {
			uncaughtExceptionHandler := DefaultUncaughtExceptionHandler()
			if uncaughtExceptionHandler != nil {
				uncaughtExceptionHandler(e)
			} else {
				fmt.Fprintf(os.Stderr, "Unhandled exception: %s\n", PrintStackTrace(e))
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
	func ensureCacheDirectory(path string) {
		defer err.Catch(func(e any) {
			if errVal, ok := e.(error); ok && errors.Is(errVal, fs.ErrExist) {
				slog.Info("cache directory already exists " + path)
				return
			}

			panic(err.NewRuntimeExceptionFrom("cannot prepare cache directory "+path, e))
		})

		createDirectory(path) // may panic
	}
*/
func Catch(handler ...func(any)) {
	if e := recover(); e != nil {
		for _, handler := range handler {
			handler(e)
		}
	}
}

/*
Repanic intercepts panic, runs failure-only logic, and always continues propagation.

Unlike ordinary defer, the handler runs only during panic-driven unwind.
The handler may rollback state, add context, or replace the panic value.

Use for compensating actions or failure enrichment before escalation.

Example:

	func writeFile(path string, data []byte) {
		defer err.Repanic(func(e any) {
			_ = os.Remove(path) // remove partial file only on failure
			panic(err.NewRuntimeExceptionFrom("cannot write file " + path, e)) // optional
		})

		createFile(path)
		writeHeader(path, data) // may panic
		writeBody(path, data) // may panic
	}
*/
func Repanic(handler ...func(any)) {
	if e := recover(); e != nil {
		for _, handler := range handler {
			handler(e)
		}
		if _, ok := e.(interface{ StackTrace() []uintptr }); ok {
			panic(e)
		}
		panic(NewRuntimeExceptionFrom("Unhandled exception", e))
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
AsAny checks whether err matches at least one target type using errors.As.

Use it for Java-like multi-catch checks.

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

	defer err.Catch(func(e any) {
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
		skip = skipFrames[0] + 2
	}
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	return append([]uintptr(nil), pcs[:n]...)
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
