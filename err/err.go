package err

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
)

type UncaughtExceptionHandler func(any)

var defaultUncaughtExceptionHandler atomic.Value

/*
To get root cause stack trace of the runtime error, like nil pointer dereference, there is no other way but let stack unwind and kill the app.

Set environment variable with comma separated list of snippets to look for in root cause error to skip recover.

Readonly. NOT for production.

	set ERR_RECOVER_DISABLED=runtime error
*/
var ERR_RECOVER_DISABLED = os.Getenv("ERR_RECOVER_DISABLED")
var errRecoverDisabledList []string

const catchFunction = "github.com/go-errr/go/err.Catch"

var hiddenFrames = map[string]struct{}{
	"runtime.goexit":  {},
	"runtime.main":    {},
	"runtime.gopanic": {},
	catchFunction:     {},
}

func init() {
	if len(ERR_RECOVER_DISABLED) > 0 {
		delim := regexp.MustCompile(",")
		for _, snippet := range delim.Split(ERR_RECOVER_DISABLED, -1) {
			errRecoverDisabledList = append(errRecoverDisabledList, strings.TrimSpace(snippet))
		}
	}
}

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
		if recoverDisabled(e) {
			panic(e)
		}
		if len(handler) == 0 {
			uncaughtExceptionHandler := DefaultUncaughtExceptionHandler()
			if uncaughtExceptionHandler != nil {
				uncaughtExceptionHandler(e)
			} else {
				fmt.Fprintf(os.Stderr, "panic: %s\n", PrintStackTrace(e))
			}
			return
		}
		assert(len(handler) <= 1, "Multiple handlers are not supported in Recover")
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
		if recoverDisabled(e) {
			panic(e)
		}
		assert(len(handler) <= 1, "Multiple handlers are not supported in Catch")
		for _, handler := range handler {
			handler(e)
		}
	}
}

/*
Assert verifies that the provided error is nil.

If e is not nil, Assert panics with an AssertionError wrapping the original
error and a formatted message.

Example:

	err.Assert(os.MkdirAll(workDir, 0755), "Cannot create %s", workDir)
	err.Assert(file.Close(), "Cannot close file %s", file.Name())
*/
func Assert(e error, format string, args ...any) {
	if e != nil {
		panic(NewAssertionErrorFrom(fmt.Sprintf(format, args...), e))
	}
}

func assert(expression bool, format string, args ...any) {
	if !expression {
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
	stack := make([]uintptr, 64)
	for {
		n := runtime.Callers(skip, stack)
		if n < len(stack) {
			return append([]uintptr(nil), stack[:n]...)
		}
		stack = make([]uintptr, len(stack)*2)
	}
}

func PrintStackTrace(e any) string {
	if e == nil {
		return ""
	}

	var b strings.Builder
	root, ok := e.(error)
	if !ok {
		fmt.Fprintf(&b, "%s\n", e)
		writeFrames(&b, logicalFrames(StackTrace(1)))
		return strings.TrimRight(b.String(), "\n")
	}

	var chain []error
	var framesChain [][]runtime.Frame
	for cur := root; cur != nil; cur = errors.Unwrap(cur) {
		chain = append(chain, cur)
		framesChain = append(framesChain, logicalFrames(stackTraceOf(cur)))
	}

	for i, err := range chain {
		if i == 0 {
			fmt.Fprintf(&b, "%T: %s\n", err, err.Error())
			frames := framesChain[i]
			if len(frames) == 0 {
				frames = logicalFrames(StackTrace(1))
			}
			writeFrames(&b, frames)
		} else {
			fmt.Fprintf(&b, "Caused by: %T: %s\n", err, err.Error())
			frames := framesChain[i]
			if len(frames) == 0 {
				continue
			}
			parentFrames := framesChain[i-1]
			common := commonTailFrames(parentFrames, frames)
			writeFrames(&b, frames[:len(frames)-common])
			if common > 0 {
				fmt.Fprintf(&b, "\t... %d common frames omitted\n", common)
			}
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func stackTraceOf(e error) []uintptr {
	type stackTracer interface {
		StackTrace() []uintptr
	}
	if st, ok := e.(stackTracer); ok {
		return st.StackTrace()
	}
	return nil
}

func logicalFrames(stack []uintptr) []runtime.Frame {
	if len(stack) == 0 {
		return nil
	}
	frames := framesOf(stack)
	frames = trimCatchCauseSegment(frames)
	result := make([]runtime.Frame, 0, len(frames))
	for _, frame := range frames {
		if _, ok := hiddenFrames[frame.Function]; !ok {
			result = append(result, frame)
		}
	}
	return result
}

func framesOf(stack []uintptr) []runtime.Frame {
	frames := runtime.CallersFrames(stack)
	result := make([]runtime.Frame, 0, len(stack))
	for {
		frame, more := frames.Next()
		result = append(result, frame)
		if !more {
			break
		}
	}
	return result
}

func trimCatchCauseSegment(frames []runtime.Frame) []runtime.Frame {
	catchIdx := indexOfFunction(frames, catchFunction, 0)
	if catchIdx <= 0 {
		return frames
	}

	handlerFn := frames[catchIdx-1].Function
	ownerFn, ok := enclosingOfDeferredHandler(handlerFn)
	if !ok {
		return frames
	}

	ownerIdx := indexOfFunction(frames, ownerFn, catchIdx+1)
	if ownerIdx < 0 {
		return frames
	}

	var res []runtime.Frame
	res = append(res, frames[:catchIdx+2]...)
	res = append(res, frames[ownerIdx:]...)
	return res
}

func indexOfFunction(frames []runtime.Frame, fn string, startIdx int) int {
	for i := startIdx; i < len(frames); i++ {
		if frames[i].Function == fn {
			return i
		}
	}
	return -1
}

func enclosingOfDeferredHandler(fn string) (string, bool) {
	idx := strings.LastIndex(fn, ".func")
	if idx < 0 {
		return "", false
	}
	return fn[:idx], true
}

func commonTailFrames(parent []runtime.Frame, child []runtime.Frame) int {
	n := 0
	i := len(parent) - 1
	j := len(child) - 1
	for i >= 0 && j >= 0 && parent[i] == child[j] {
		n++
		i--
		j--
	}
	return n
}

func writeFrames(b *strings.Builder, frames []runtime.Frame) {
	for _, frame := range frames {
		fmt.Fprintf(b, "\tat %s (%s:%d)\n", frame.Function, frame.File, frame.Line)
	}
}

func recoverDisabled(e any) bool {
	if len(errRecoverDisabledList) == 0 {
		return false
	}
	message := rootCauseMessage(e)
	for _, snippet := range errRecoverDisabledList {
		if strings.Contains(message, snippet) {
			return true
		}
	}
	return false
}

func rootCauseMessage(e any) string {
	var err error
	switch v := e.(type) {
	case error:
		err = v
	default:
		return fmt.Sprint(e)
	}

	for {
		cause := errors.Unwrap(err)
		if cause == nil {
			break
		}
		err = cause
	}
	return err.Error()
}

func DefaultUncaughtExceptionHandler() UncaughtExceptionHandler {
	handler, _ := defaultUncaughtExceptionHandler.Load().(UncaughtExceptionHandler)
	return handler
}

func SetDefaultUncaughtExceptionHandler(handler UncaughtExceptionHandler) {
	defaultUncaughtExceptionHandler.Store(handler)
}
