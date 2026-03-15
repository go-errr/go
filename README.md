# Exception Handling for Go

This package provides a **three-polar panic control model** for Go applications.  
It allows developers to **contain failures, execute recovery workflows, enrich errors, and control stack unwind behavior** in concurrent and service-oriented systems.  

The model is built around three primitives:

- **Recover** — contain failure and stop panic propagation  
- **Catch** — intercept failure and decide how unwind should proceed  
- **Repanic** — run failure logic and continue panic propagation  

The goal is to make **failure handling explicit, structured, and safe for concurrent execution**.

### Why

In large Go systems:

- background goroutines must not crash the whole service  
- failure often requires **compensating business logic**  
- rollback code may itself fail  
- some failures should not interrupt the flow  
- some failures must propagate with additional context  

Standard `recover()` is a low-level mechanism.  
This package introduces **execution policy semantics** on top of it.

### err.Recover

Recover intercepts a panic, executes recovery logic, and always stops panic propagation.  
Use it at execution boundaries where a panic must never escape, even if the recovery
logic itself fails. For example, at the beginning of a new goroutine.  
The handler may perform logging, rollback, cleanup, or business state updates.
If the handler itself panics, stack unwinding is still stopped.  
If no handler is provided, the default uncaught exception handler is used;
otherwise, the stack trace is printed to stderr.

Example:

```go
func processImportAsync(jobId string) {
  go func() {
    defer err.Recover(func(e any) {
      markJobFailed(jobId) // may panic
      slog.Error("import worker failed. " + err.PrintStackTrace(e))
    })

    processImport(jobId) // may panic
  }()
}
```

### err.Catch

Catch intercepts panic and lets the developer decide how unwind should proceed.  
The handler may run compensating logic, swallow the panic, or continue unwind
with the same or a wrapped value.  
Use when failure handling is part of business logic.

Example:

```go
func ensureCacheDirectory(path string) {
  defer err.Catch(func(e any) {
    if err.Is(e, fs.ErrExist) {
      slog.Info("cache directory already exists " + path)
    } else {
      panic(err.NewRuntimeExceptionFrom("cannot prepare cache directory " + path, e))
    }
  })

  createDirectory(path) // may panic
}
```

### err.Repanic

Repanic intercepts panic, runs failure-only logic, and always continues propagation.  
Unlike ordinary defer, the handler runs only during panic-driven unwind.
The handler may rollback state, add context, or replace the panic value.  
Use for compensating actions or failure enrichment before escalation.

Example:

```go
func writeFile(path string, data []byte) {
  defer err.Repanic(func(e any) {
    _ = os.Remove(path) // remove partial file only on failure
    panic(err.NewRuntimeExceptionFrom("cannot write file " + path, e)) // optional
  })

  createFile(path)
  writeHeader(path, data) // may panic
  writeBody(path, data) // may panic
}
```

## Stack Traces and Exception Hierarchy

The base type **AbstractThrowable** supports:

- failure message
- wrapped cause chain
- captured stack trace
- compatibility with Go `errors.Is` / `errors.As`

Stack traces can be printed in a clear and readable form, helping operators
understand the failure impact without losing diagnostic depth.

    2026/03/14 02:21:23 ERROR Context refresh failed. *concurrent.ExecutionException: Some error
            at github.com/go-beans/go/concurrent.(*Executor[...]).Submit (D:/dev/go-beans/concurrent/Executor.go:82)
            at github.com/go-beans/go/ioc.(*ApplicationContext).startLifecycleBeans (D:/dev/go-beans/ioc/ApplicationContext.go:205)
            at github.com/go-beans/go/ioc.(*ApplicationContext).Refresh (D:/dev/go-beans/ioc/ApplicationContext.go:172)
            at github.com/go-beans/go/ioc.(*ApplicationContext).Run (D:/dev/go-beans/ioc/ApplicationContext.go:264)
            at github.com/go-beans/go/ioc.Run (D:/dev/go-beans/ioc/ioc.go:79)
            at main.main (D:/dev/playground/cmd/app/main.go:27)
    Caused by: *err.RuntimeException: Some error
            at playground/internal/app.(*Service2).Start (D:/dev/playground/internal/app/Service2.go:24)
            at github.com/go-beans/go/ioc.(*ApplicationContext).startLifecycleBeans.func1 (D:/dev/go-beans/ioc/ApplicationContext.go:206)
            at github.com/go-beans/go/concurrent.(*Executor[...]).Submit.func1 (D:/dev/go-beans/concurrent/Executor.go:87)
            at github.com/go-beans/go/concurrent.NewExecutor[...].func1 (D:/dev/go-beans/concurrent/Executor.go:72)

### Error vs Exception

The hierarchy follows a conceptual model similar to Java:

- **AbstractError** represents serious failures that a reasonable application
  should not try to handle locally. These failures are expected to propagate
  to a top-level boundary such as an HTTP error interceptor.

- **AbstractException** represents conditions that application code may
  reasonably want to catch and handle as part of normal business logic.

Developers are free to define their own hierarchy on top of these base types.  
For example, a **ClientError** may be introduced to represent invalid user input.
Such errors should propagate through service layers and be detected at the
HTTP boundary using `errors.As`.

### HTTP Boundary Handling

At the top-level error interceptor:

- If a **client error** is detected in the exception chain:
  - a clear root-cause message is returned to the user
  - stack trace (or message) is logged with **warning severity**
  - HTTP status **400** is returned

- If no client error is found:
  - the failure is treated as a **server error**
  - stack trace is logged with **error severity**
  - the top-level message is returned to the client
  - HTTP status **500** is returned

This model allows:

- deep diagnostic stack traces for operators
- clean and actionable messages for users
- consistent propagation of failure semantics across service layers

### RuntimeException

A ready-to-use general purpose exception that captures stack trace.  
Useful when unexpected failures occur to panic with stack trace attached or
additional context added before propagating the panic.

### InterruptedException

`InterruptedException` represents **controlled interruption of execution** such as:

- service shutdown
- canceled context
- broken network connection
- closed pipe or stream

Interruption is not always treated as an application failure, but it still
travels through the stack and activates compensating logic on each level.  
This is important when multiple layers own different pieces of state and each
layer must rollback its own part before unwind stops.  
The helper `err.Interrupted(...)` allows application code to recognize such
conditions and decide whether to propagate them further.

Example:

```go
func doHeavyComputation(file os.File) {
	defer err.Catch(func(e any) {
		if err.Interrupted(e) {
			updateProcessingStatus(file.Name(), "CANCELLED")
			panic(e) // rethrow interruption so higher levels can rollback too
		}

		updateProcessingStatus(file.Name(), "FAILED")
		panic(err.NewRuntimeExceptionFrom("file processing failed "+file.Name(), e))
	})

	processLineByLine(file) // may panic
}
```
