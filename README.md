# Exception Handling for Go

### Philosophy

Failure handling is expressed through a single deferred handler that:

- intercepts panic
- classifies failure
- decides propagation policy

Unrecognized failures normally continue to unwind the stack.  
This leads to readable and maintainable failure control flow:

```go
defer err.Catch(func(e any) {
  switch {
  case err.Interrupted(e):
    updateProcessingStatus(file.Name(), "CANCELLED")
    panic(e) // rethrow interruption so higher levels can rollback too

  case err.As2[*os.PathError, *net.OpError](e):
    slog.Warn("external resource unavailable")
    scheduleRetry()

  default:
    updateProcessingStatus(file.Name(), "FAILED")
    panic(e)
  }
})
```

**Catch** intercepts panic and lets the developer decide how unwind should proceed.
The handler may run compensating logic, swallow the panic, or continue unwind
with the same or a wrapped value.
Use when failure handling is part of business logic.

Example:

```go
func processImport(jobId string) {
  // do rollback/cleanup
  defer err.Catch(func(e any) {
    markJobFailed(jobId) // may panic
    panic(err.NewRuntimeExceptionFrom(fmt.Sprintf("Job %s failed", jodId), e))
  })

  processImport(jobId) // may panic
  markJobSucceded(jobId) // may panic
}
```

**Recover** intercepts a panic, executes recovery logic, and always stops panic propagation.  
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
    // safe for go-routine
    defer err.Recover(func(e any) {
      markJobFailed(jobId) // may panic
      slog.Error(fmt.Sprintf("Job %s failed. %s", jodId, err.PrintStackTrace(e)))
    })

    processImport(jobId) // may panic
    markJobSucceded(jobId) // may panic
  }()
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

```
2026/05/17 14:39:26 ERROR Context run failed. *err.RuntimeException: Error creating bean *app.ApplicationRunner1 [singleton lazy ApplicationRunner]
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance.func1 (D:/dev/go-beans/ioc/ApplicationContext.go:140)
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance (D:/dev/go-beans/ioc/ApplicationContext.go:149)
        at github.com/go-beans/go/ioc.(*ApplicationContext).Bean (D:/dev/go-beans/ioc/ApplicationContext.go:133)
        at github.com/go-beans/go/ioc.(*InjectQualifier[...]).doResolve (D:/dev/go-beans/ioc/InjectQualifier.go:60)
        at github.com/go-beans/go/ioc.(*InjectQualifier[...]).resolve.func1.1 (D:/dev/go-beans/ioc/InjectQualifier.go:52)
        at sync.(*Once).doSlow (C:/Program Files/Go/src/sync/once.go:78)
        at sync.(*Once).Do (C:/Program Files/Go/src/sync/once.go:69)
        at github.com/go-beans/go/ioc.(*InjectQualifier[...]).resolve.func1 (D:/dev/go-beans/ioc/InjectQualifier.go:51)
        at github.com/go-beans/go/ioc.(*BeanDefinitionImpl[...]).instantiate.injectBeansAny.func2 (D:/dev/go-beans/ioc/ioc.go:84)
        at github.com/go-external-config/go/util/reflects.ForEachTaggedField (D:/dev/go-external-config/util/reflects/reflects.go:39)
        at github.com/go-beans/go/ioc.injectBeansAny (D:/dev/go-beans/ioc/ioc.go:76)
        at github.com/go-beans/go/ioc.(*BeanDefinitionImpl[...]).instantiate (D:/dev/go-beans/ioc/BeanDefinition.go:240)
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance.func2 (D:/dev/go-beans/ioc/ApplicationContext.go:152)
        at github.com/go-external-config/go/util/concurrent.Synchronized (D:/dev/go-external-config/util/concurrent/concurrent.go:11)
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance (D:/dev/go-beans/ioc/ApplicationContext.go:149)
        at github.com/go-beans/go/ioc.(*ApplicationContext).orderedBeanInstances.func1 (D:/dev/go-beans/ioc/ApplicationContext.go:268)
        at github.com/go-beans/go/ioc.(*ApplicationContext).foreachBeanDefinition (D:/dev/go-beans/ioc/ApplicationContext.go:427)
        at github.com/go-beans/go/ioc.(*ApplicationContext).orderedBeanInstances (D:/dev/go-beans/ioc/ApplicationContext.go:266)
        at github.com/go-beans/go/ioc.(*ApplicationContext).executeApplicationRunnerBeans (D:/dev/go-beans/ioc/ApplicationContext.go:320)
        at github.com/go-beans/go/ioc.(*ApplicationContext).Run (D:/dev/go-beans/ioc/ApplicationContext.go:315)
        at github.com/go-beans/go/ioc.Run (D:/dev/go-beans/ioc/ioc.go:135)
        at main.main (D:/dev/playground/cmd/app/main.go:33)
Caused by: *err.RuntimeException: Cannot inject dependency into field 'service1' of type *app.Service1
        at github.com/go-beans/go/ioc.(*ApplicationContext).Bean.func1 (D:/dev/go-beans/ioc/ApplicationContext.go:85)
        ... 20 common frames omitted
Caused by: *err.RuntimeException: Error creating bean *app.Service1 [singleton lazy]
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance.func1 (D:/dev/go-beans/ioc/ApplicationContext.go:140)
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance (D:/dev/go-beans/ioc/ApplicationContext.go:149)
        ... 20 common frames omitted
Caused by: *err.RuntimeException: Cannot bind configuration value '${vault.prod/db#password}' to field 'dbPass'
        at github.com/go-beans/go/ioc.(*BeanDefinitionImpl[...]).instantiate.BindPropertiesAny.func1.1 (D:/dev/go-external-config/env/env.go:58)
        at github.com/go-external-config/go/util/optional.(*Optional[...]).panicIfEmpty (D:/dev/go-external-config/util/optional/Optional.go:80)
        at github.com/go-external-config/go/util/optional.(*Optional[...]).OrElsePanic (D:/dev/go-external-config/util/optional/Optional.go:74)
        at github.com/go-external-config/vault/env.(*VaultPropertySource).getSecretValue (D:/dev/go-external-config-vault/env/VaultPropertySource.go:84)
        at github.com/go-external-config/vault/env.(*VaultPropertySource).resolveVaultProperty (D:/dev/go-external-config-vault/env/VaultPropertySource.go:80)
        at github.com/go-external-config/vault/env.(*VaultPropertySource).Property (D:/dev/go-external-config-vault/env/VaultPropertySource.go:60)
        at github.com/go-external-config/go/env.(*Environment).lookupRawProperty (D:/dev/go-external-config/env/Environment.go:81)
        at github.com/go-external-config/go/env.(*ExprProcessor).Resolve (D:/dev/go-external-config/env/ExprProcessor.go:64)
        at github.com/go-external-config/go/env.ExprProcessorOf.(*PatternProcessor).OverrideResolve.func1 (D:/dev/go-external-config/util/regex/PatternProcessor.go:68)
        at github.com/go-external-config/go/util/regex.(*PatternProcessor).ProcessRecursive (D:/dev/go-external-config/util/regex/PatternProcessor.go:42)
        at github.com/go-external-config/go/util/regex.(*PatternProcessor).Process (D:/dev/go-external-config/util/regex/PatternProcessor.go:24)
        at github.com/go-external-config/go/env.(*Environment).ResolveRequiredPlaceholders (D:/dev/go-external-config/env/Environment.go:89)
        at github.com/go-beans/go/ioc.(*BeanDefinitionImpl[...]).instantiate.BindPropertiesAny.func1 (D:/dev/go-external-config/env/env.go:60)
        at github.com/go-external-config/go/util/reflects.ForEachTaggedField (D:/dev/go-external-config/util/reflects/reflects.go:39)
        at github.com/go-external-config/go/env.BindPropertiesAny (D:/dev/go-external-config/env/env.go:56)
        at github.com/go-beans/go/ioc.(*BeanDefinitionImpl[...]).instantiate (D:/dev/go-beans/ioc/BeanDefinition.go:239)
        at github.com/go-beans/go/ioc.(*ApplicationContext).beanInstance.func2 (D:/dev/go-beans/ioc/ApplicationContext.go:152)
        at github.com/go-external-config/go/util/concurrent.Synchronized (D:/dev/go-external-config/util/concurrent/concurrent.go:11)
        ... 21 common frames omitted
Caused by: *err.RuntimeException: Unable to get prod/db
        at github.com/go-external-config/go/util/optional.(*Optional[...]).panicIfEmpty (D:/dev/go-external-config/util/optional/Optional.go:80)
        ... 37 common frames omitted
Caused by: *fmt.wrapError: error encountered while reading secret at secret/data/prod/db: Get "http://127.0.0.1:8200/v1/secret/data/prod/db": dial tcp 127.0.0.1:8200: connectex: No connection could be made because the target machine actively refused it.
Caused by: *url.Error: Get "http://127.0.0.1:8200/v1/secret/data/prod/db": dial tcp 127.0.0.1:8200: connectex: No connection could be made because the target machine actively refused it.
Caused by: *net.OpError: dial tcp 127.0.0.1:8200: connectex: No connection could be made because the target machine actively refused it.
Caused by: *os.SyscallError: connectex: No connection could be made because the target machine actively refused it.
Caused by: syscall.Errno: No connection could be made because the target machine actively refused it.
```

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
    panic(err.NewRuntimeExceptionFrom("file processing failed " + file.Name(), e))
  })

  processLineByLine(file) // may panic
}
```
