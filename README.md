# Separating Error-Handling Code from Regular Code

Go's explicit error returns make failures visible at each operation. This works well for simple code, but when a method performs several operations that can fail, error propagation can obscure the regular flow of the method.

Consider processing files in a loop. Opening, reading, parsing, storing, and publishing each file may fail:

```go
func processFiles(paths []string) {
	for _, path := range paths {
		if e := processFile(path); e != nil {
			if errors.Is(e, context.Canceled) {
				return
			}
			var parseError *ParseError
			if errors.As(e, &parseError) {
				log4g.Error("Cannot process file {}: {}", path, err.PrintStackTrace(e))
				continue
			}
			log4g.Error("Failed to process file {}: {}", path, err.PrintStackTrace(e))
		}
	}
}

func processFile(path string) error {
	file, e := os.Open(path)
	if e != nil {
		return fmt.Errorf("Cannot open file %s: %w", path, e)
	}
	defer file.Close()

	content, e := io.ReadAll(file)
	if e != nil {
		return fmt.Errorf("Cannot read file %s: %w", path, e)
	}

	data, e := parse(content)
	if e != nil {
		return fmt.Errorf("Cannot parse file %s: %w", path, e)
	}

	if e = repository.Save(data); e != nil {
		return fmt.Errorf("Cannot save data: %w", e)
	}
	if e = publisher.Publish(data); e != nil {
		return fmt.Errorf("Cannot publish data: %w", e)
	}
	return nil
}
```

The regular flow of `processFile` is simple:

```text
open → read → parse → save → publish
```

but it is interleaved with repetitive error checking and propagation. The caller then needs another layer of error classification to decide whether processing should continue.

With `err.Catch`, failure handling can be separated from the regular processing code:

```go
func processFiles(paths []string) {
	for _, path := range paths {
		processFile(path)
	}
}

func processFile(path string) {
	defer err.Catch(func(e any) {
		switch {
		case err.Interrupted(e):
			panic(e)
		case err.As1[*ParseError](e):
			log4g.Error("Cannot process file {}: {}", path, err.PrintStackTrace(e))
		default:
			log4g.Error("Failed to process file {}: {}", path, err.PrintStackTrace(e))
		}
	})

	file := optional.OfCommaErr(os.Open(path)).OrElsePanic("Cannot open file %s", path)
	defer file.Close()
	content := optional.OfCommaErr(io.ReadAll(file)).OrElsePanic("Cannot read file %s", path)
	data := optional.OfCommaErr(parse(content)).OrElsePanic("Cannot parse file %s", path)
	err.Assert(repository.Save(data), "Cannot save data")
	err.Assert(publisher.Publish(data), "Cannot publish data")
}
```

The main body now describes the successful path directly. Failure handling is concentrated in one place and expresses the actual policy of the method:

* interruption is rethrown so that higher execution levels can stop processing;
* known failures can be handled with a specific message;
* unexpected failures are logged with a general error and consumed, allowing the loop to continue with the next file.

This does not eliminate error handling. Failures are still detected, classified, logged, or propagated as appropriate. The difference is that error-handling policy no longer obscures the regular control flow.

# err.Try

**Try** executes an action within a local try/catch block.

A panic raised by the action is passed to the handler. Once handled, execution continues after the `Try` call.

Use `Try` when failure handling applies to a specific operation or block inside a function.

```go
for message := range messages {
	err.Try(func() {
		processMessage(message)
	}, func(e any) {
		logProcessingError(e)
	})
}
```

# err.Catch

`Catch` provides a function-wide catch.

Because `Catch` is deferred, it handles a panic escaping anywhere from the remaining execution of the current function. The handler may run compensating logic, swallow the panic, or continue unwind with the same or a wrapped value.

Use `Catch` when failure handling applies to the function as a whole.

```go
func processImport(jobId string) {
	defer err.Catch(func(e any) {
		markJobFailed(jobId)
		panic(err.NewRuntimeExceptionFrom(fmt.Sprintf("Job %s failed", jobId), e))
	})
	processImportFile(jobId)
	markJobSucceeded(jobId)
}
```

This pattern is useful when the current function owns state that must be updated or released before the failure propagates. The handler performs the compensating logic and then rethrows the same or a wrapped failure so the enclosing request or operation still fails.

# err.Recover

`Recover` provides a goroutine-level recovery boundary.

Use `Recover` at the entry point of a goroutine where a panic must never escape. Recovery logic itself is protected, so a failure in the recovery handler does not cause another panic to escape the boundary.

```go id="1dwqze"
go func() {
	defer err.Recover()
	processMessages()
}()
```

When no handler is provided, `Recover` delegates the panic to the default uncaught exception handler. The handler is process-wide and can be configured once during application initialization:

```go id="bnqk4d"
err.SetDefaultUncaughtExceptionHandler(func(e any) {
	log4g.Error("Uncaught exception: {}", err.PrintStackTrace(e))
})
```

If no default uncaught exception handler has been configured, `Recover` prints the panic and its stack trace to `stderr`.

An explicit handler can be provided when a particular goroutine requires additional recovery logic:

```go id="ymu48m"
go func() {
	defer err.Recover(func(e any) {
		log4g.Error("Backup job failed: {}", err.PrintStackTrace(e))
		markBackupFailed()
	})
	runBackup()
}()
```

In all cases, a panic handled by `Recover` does not escape the recovery boundary.


# Stack Traces and Exception Hierarchy

The base type `AbstractThrowable` supports:

- failure message
- wrapped cause chain
- captured stack trace
- compatibility with Go `errors.Is` / `errors.As`

Stack traces can be printed in a clear and readable form, helping operators
understand the failure impact without losing diagnostic depth.

```
D:\dev\playground>go run ./cmd/app
LOG4G INFO  Loaded configuration from config/log4g.yaml
2026-09-04 18:45:48.295 INFO  github.com/go-external-config/go/env/Environment:215 - Loaded configuration from config/application.yaml
2026-09-04 18:45:48.296 INFO  github.com/go-external-config/go/env/Environment:215 - Loaded configuration from config/application-live.properties
2026-09-04 18:45:48.301 INFO  github.com/go-beans/go/ioc:56 - Starting with PID 41068
2026-09-04 18:45:48.317 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.Service1 [singleton Lifecycle]
2026-09-04 18:45:48.323 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.Service2 [singleton lazy Lifecycle]
2026-09-04 18:45:48.324 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.Service3 [singleton Lifecycle]
2026-09-04 18:45:48.324 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.Service4 [singleton service4]
2026-09-04 18:45:48.325 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.Service5 [singleton service5]
2026-09-04 18:45:48.326 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.ApplicationRunner1 [singleton lazy ApplicationRunner]
2026-09-04 18:45:48.327 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.ApplicationRunner2 [singleton ApplicationRunner]
2026-09-04 18:45:48.331 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *app.ApplicationRunner3 [singleton ApplicationRunner]
2026-09-04 18:45:48.332 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *concurrent.Executor[*github.com/redis/go-redis/v9.IntCmd] [singleton publishExecutor]
2026-09-04 18:45:48.335 DEBUG github.com/go-beans/go/ioc/ApplicationContext:81 - Registered *cron.Cron [singleton]
2026-09-04 18:45:48.340 ERROR github.com/go-beans/go/ioc/ApplicationContext:345 - Context run failed. *err.RuntimeException: Error creating bean *app.Service1 [singleton Lifecycle]
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance.func1 (D:/dev/go-beans/ioc/ApplicationContext.go:145)
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance (D:/dev/go-beans/ioc/ApplicationContext.go:154)
        at github.com/go-beans/go/ioc.(*ApplicationContext).initializeBeans.func2 (D:/dev/go-beans/ioc/ApplicationContext.go:193)
        at github.com/go-beans/go/ioc.(*ApplicationContext).foreachBeanDefinition (D:/dev/go-beans/ioc/ApplicationContext.go:467)
        at github.com/go-beans/go/ioc.(*ApplicationContext).initializeBeans (D:/dev/go-beans/ioc/ApplicationContext.go:190)
        at github.com/go-beans/go/ioc.(*ApplicationContext).doRefresh (D:/dev/go-beans/ioc/ApplicationContext.go:181)
        at github.com/go-beans/go/ioc.(*ApplicationContext).run (D:/dev/go-beans/ioc/ApplicationContext.go:293)
        at github.com/go-beans/go/ioc.Run (D:/dev/go-beans/ioc/ioc.go:288)
        at main.main (D:/dev/playground/cmd/app/main.go:49)
Caused by: *err.RuntimeException: Cannot bind configuration value '${db.pass}' to field 'dbPass'
        at github.com/go-external-config/go/env.BindPropertiesAny.func1.1 (D:/dev/go-external-config/env/env.go:71)
        at github.com/go-external-config/go/env.BindPropertiesAny (D:/dev/go-external-config/env/env.go:69)
        at github.com/go-beans/go/ioc.(*BeanDefinitionImpl[...]).instantiate (D:/dev/go-beans/ioc/BeanDefinition.go:262)
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance.func2 (D:/dev/go-beans/ioc/ApplicationContext.go:157)
        at github.com/go-jang/go/util/concurrent.Synchronized (D:/dev/go-jang/util/concurrent/concurrent.go:11)
        ... 8 common frames omitted
Caused by: *err.RuntimeException: Cannot get Consul property app/db/password
        at github.com/go-external-config/consul/env.(*ConsulPropertySource).getPropertyValue (D:/dev/go-external-config-consul/env/ConsulPropertySource.go:67)
        at github.com/go-external-config/consul/env.(*ConsulPropertySource).Property (D:/dev/go-external-config-consul/env/ConsulPropertySource.go:58)
        at github.com/go-external-config/go/env.(*Environment).lookupRawProperty (D:/dev/go-external-config/env/Environment.go:103)
        at github.com/go-external-config/go/env.(*ExprProcessor).Resolve (D:/dev/go-external-config/env/ExprProcessor.go:62)
        at github.com/go-jang/go/util/regex.(*PatternProcessor).OverrideResolve.func1 (D:/dev/go-jang/util/regex/PatternProcessor.go:73)
        at github.com/go-jang/go/util/regex.(*PatternProcessor).ProcessRecursive (D:/dev/go-jang/util/regex/PatternProcessor.go:47)
        at github.com/go-jang/go/util/regex.(*PatternProcessor).Process (D:/dev/go-jang/util/regex/PatternProcessor.go:24)
        at github.com/go-external-config/go/env.(*Environment).resolveRequiredPlaceholders (D:/dev/go-external-config/env/Environment.go:115)
        at github.com/go-external-config/go/env.ResolveRequiredPlaceholders (D:/dev/go-external-config/env/env.go:36)
        at github.com/go-external-config/go/env.BindPropertiesAny.func1 (D:/dev/go-external-config/env/env.go:73)
        at github.com/go-jang/go/lang/reflect.ForEachTaggedField (D:/dev/go-jang/lang/reflect/reflect.go:39)
        ... 12 common frames omitted
Caused by: *url.Error: Get "http://127.0.0.1:8500/v1/kv/app/db/password"
Caused by: *net.OpError: dial tcp 127.0.0.1:8500
Caused by: *os.SyscallError: connectex
Caused by: syscall.Errno: No connection could be made because the target machine actively refused it.
2026-09-04 18:45:48.342 INFO  github.com/go-beans/go/ioc/ApplicationContext:313 - Closing context with 1 running services
2026-09-04 18:45:48.343 INFO  github.com/go-beans/go/ioc/ApplicationContext:320 - Context closed in 695.8µs, uptime 25.9243ms
exit status 1
```

# Error vs Exception

The hierarchy follows a conceptual model similar to Java:

- `AbstractError` represents serious failures that a reasonable application
  should not try to handle locally. These failures are expected to propagate
  to a top-level boundary such as an HTTP error interceptor.

- `AbstractException` represents conditions that application code may
  reasonably want to catch and handle as part of normal business logic.

Developers are free to define their own hierarchy on top of these base types.  
For example, a `ClientError` may be introduced to represent invalid user input.
Such errors should propagate through service layers and be detected at the
HTTP boundary using `errors.As`.

# HTTP Boundary Handling

At the top-level error interceptor:

- If a `client error` is detected in the exception chain:
  - a clear root-cause message is returned to the user
  - stack trace (or message) is logged with `warning severity`
  - HTTP status `400` is returned

- If no client error is found:
  - the failure is treated as a `server error`
  - stack trace is logged with `error severity`
  - the top-level message is returned to the client
  - HTTP status `500` is returned

This model allows:

- deep diagnostic stack traces for operators
- clean and actionable messages for users
- consistent propagation of failure semantics across service layers

# RuntimeException

A ready-to-use general purpose exception that captures stack trace.  
Useful when unexpected failures occur to panic with stack trace attached or
additional context added before propagating the panic.

# InterruptedException

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
    panic(err.NewRuntimeExceptionFrom("file processing failed " + file.Name(), e))
  })

  processLineByLine(file) // may panic
}
```
