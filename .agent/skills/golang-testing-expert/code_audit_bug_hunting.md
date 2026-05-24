# Code Auditing & Bug Hunting in Golang

## Code Review Methodology: Systematic Audit Checklist

### 1. Concurrency & Data Race Bugs
- **Shared Mutable State:** Searching for global variables or struct fields accessed by multiple goroutines without proper `sync.Mutex` or `sync.RWMutex` protection.
- **Goroutine Leaks:** Identifying goroutines that are launched but never terminated — especially goroutines blocked on channels that are never closed or receive operations that never fire.
- **Channel Misuse:** Spotting send-on-closed-channel panics, double-close panics, and deadlocks from goroutines blocking indefinitely on unbuffered channels.
- **Detection Tool:** `go test -race ./...` to catch races dynamically; static code review to find them structurally.

### 2. Error Handling Gaps
- **Silently Ignored Errors:** Finding patterns like `result, _ := someFunction()` where the error is discarded. This is one of the most common sources of silent failures in Go.
- **Incorrect Error Wrapping:** Errors that are returned without context, making stack traces useless for debugging (`return err` vs `return fmt.Errorf("creating user: %w", err)`).
- **Panic Without Recovery:** Identifying code paths that can `panic()` in production without a `recover()` handler, causing the entire server to crash.

### 3. Resource Leak Bugs
- **Unclosed HTTP Response Bodies:** Forgetting `defer resp.Body.Close()` after an `http.Client.Do()` call causes connection pool exhaustion.
- **Unclosed Database Rows:** Forgetting `defer rows.Close()` in database query loops.
- **Unclosed File Handles:** Missing `defer f.Close()` after `os.Open()`.
- **Context Not Cancelled:** Calling `context.WithCancel()` or `context.WithTimeout()` without `defer cancel()`, causing context leaks.

### 4. Logic & Business Rule Bugs
- **Off-by-One Errors:** Incorrect loop bounds in slice/array iteration.
- **Integer Overflow:** Arithmetic on `int` or `int32` that can exceed the maximum value silently.
- **Nil Pointer Dereference:** Accessing fields on a pointer returned from a function without a nil check.
- **Incorrect Pagination Logic:** Wrong offset/limit calculations in database queries.
- **Time Zone Bugs:** Using `time.Now()` when `time.Now().UTC()` is required for consistent timestamps.
