package err

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"
)

/*
Recover intercepts panic, runs recovery logic, and always stops panic propagation.

Use at execution boundaries where panic must never escape, even if recovery
logic itself fails.

The handler may perform logging, rollback, cleanup, or business state changes.
If the handler itself panics, fallback diagnostics are emitted, but unwind is
still stopped.

Example:

	func runImportWorkerAsync(jobId string) {
		go func() {
			defer err.Recover(func(e any) {
				markJobFailed(jobId)      // may panic
				log.Error("import worker failed", e)
			})

			processImport(jobId) // may panic
		}()
	}
*/
func Recover(handler ...func(any)) {
	if e := recover(); e != nil {
		if len(handler) == 0 {
			fmt.Fprintf(os.Stderr, "Unhandled exception: %s\n", PrintStackTrace(e))
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
				log.Info("cache directory already exists", path)
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

func Interrupted(err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}
	var ie *InterruptedException
	if errors.As(e, &ie) {
		return true
	}
	// client cancelled request, server stopped
	if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) {
		return true
	}
	// standard connection closed
	if errors.Is(e, io.EOF) || errors.Is(e, net.ErrClosed) {
		return true
	}
	// broken pipe / connection reset / connection aborted
	var errno syscall.Errno
	if errors.As(e, &errno) {
		switch errno {
		case syscall.EPIPE, syscall.ECONNRESET, syscall.ECONNABORTED:
			return true
		}
	}
	// wrapped broken pipe / connection reset / connection aborted
	var se *os.SyscallError
	if errors.As(e, &se) && (errors.Is(se.Err, syscall.EPIPE) || errors.Is(se.Err, syscall.ECONNRESET) || errors.Is(se.Err, syscall.ECONNABORTED)) {
		return true
	}
	return false
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
