# go-routine

A robust goroutine pool and task group management library for Go.

## Features

- **Goroutine Pool**: Limits the number of concurrent goroutines and provides a task queue.
- **Task Groups**: Manage a set of related tasks, wait for their completion, and propagate errors.
- **Context Awareness**: Support for context cancellation to stop tasks and group submissions.
- **Panic Resilience**: Workers and groups automatically recover from panics, ensuring stability.
- **Modern Go**: Built with Go 1.26 idioms.

## Installation

```bash
go get github.com/paulusrobin/go-routine
```

## Usage

### Goroutine Pool

The pool allows you to submit tasks that will be executed by a limited number of workers.

```go
package main

import (
    "context"
    "fmt"
    "github.com/paulusrobin/go-routine"
)

func main() {
    // Create a pool with 5 workers and a queue size of 10
    pool := goroutine.NewPool(5, 10)
    defer pool.Stop()

    // Submit a task
    err := pool.Submit(func(ctx context.Context) error {
        fmt.Println("Running task in pool")
        return nil
    })
    
    if err != nil {
        fmt.Printf("Failed to submit: %v\n", err)
    }
}
```

### Task Group

A `Group` allows you to run multiple tasks and wait for all of them to finish. If any task returns an error or panics, the entire group's context is cancelled, stopping other tasks.

```go
package main

import (
    "context"
    "fmt"
    "github.com/paulusrobin/go-routine"
    "time"
)

func main() {
    pool := goroutine.NewPool(2, 5)
    defer pool.Stop()

    // Create a group with a parent context
    group := pool.Group(context.Background())

    // Submit multiple tasks
    group.Go(func(ctx context.Context) error {
        time.Sleep(100 * time.Millisecond)
        fmt.Println("Task 1 completed")
        return nil
    })

    group.Go(func(ctx context.Context) error {
        fmt.Println("Task 2 completed")
        return nil
    })

    // Wait for all tasks to finish and get the first error if any
    if err := group.Wait(); err != nil {
        fmt.Printf("Group failed: %v\n", err)
    }
}
```

### Context Cancellation

If a task in a group fails, the group's context is cancelled. Other tasks should listen to `ctx.Done()` to stop gracefully.

```go
group.Go(func(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(time.Second):
        return nil
    }
})
```

## License

MIT
