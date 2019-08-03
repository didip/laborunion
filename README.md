[![GoDoc](https://godoc.org/github.com/didip/laborunion?status.svg)](http://godoc.org/github.com/didip/laborunion)
[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/didip/laborunion/master/LICENSE)

## laborunion

It's a worker pool library. One might say it has a simple actor pattern (blocking channel + worker pool).


## Five Minute Tutorial

```go
package main

import (
    "fmt"
    "github.com/didip/laborunion"
)

func main() {
    // 1. Configure worker pool
    pool := laborunion.New()

    pool.SetWorker(func(tasks []interface{}) error {
        for _, task := range tasks {
            fmt.Printf("Running a task: %v\n", task)
        }
        return nil
    })

    pool.SetWorkerCount(5)

    // 2. Send tasks
    for i := 0; i < 10; i++ {
        pool.GetInChan() <- i
    }
}
```

## Features

1. Able to dynamically resize number of workers.

    ```go
    pool := laborunion.New()

    pool.SetWorker(func(tasks []interface{}) error {
        for _, task := range tasks {
            fmt.Printf("Running a task: %v\n", task)
        }
    })
    pool.SetWorkerCount(5)
    pool.SetWorkerCount(10)
    ```

2. Various hooks for logging or other purposes.

    ```go
    pool := laborunion.New()

    pool.SetBeforeBatchingHook(func() {
        fmt.Printf("DEBUG: I am called before batching tasks.")
    })
    pool.SetAfterBatchingHook(func([]interfaces) {
        fmt.Printf("DEBUG: I am called after batching tasks.")
    })
    pool.SetOnNewWorker(func() {
        fmt.Printf("DEBUG: Launched a new worker.")
    })
    pool.SetOnDeleteWorker(func() {
        fmt.Printf("DEBUG: Deleted a worker.")
    })
    pool.SetOnFailedWork(func(err error) {
        fmt.Printf("ERROR: Task failed: %v", err)
    })
    pool.SetOnSuccessWork(func(task interface{}) {
        fmt.Printf("INFO: Task succeeded: %v", task)
    })
    ```

3. Configurable retries.

    ```go
    pool := laborunion.New()
    pool.SetRetries(5)
    pool.SetMaxRetryMilliseconds(20)
    ```

4. No external libraries used.


## My other Go libraries

* [Tollbooth](https://github.com/didip/tollbooth): Simple middleware to rate-limit HTTP requests.

* [Gomet](https://github.com/didip/gomet): Simple HTTP client & server long poll library for Go.

* [Stopwatch](https://github.com/didip/stopwatch): A small library to measure latency of things. Useful if you want to report latency data to Graphite.
