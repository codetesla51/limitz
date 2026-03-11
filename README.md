# Limitz

A high-performance, extensible rate limiting library for Go. Limitz provides five battle-tested rate limiting algorithms with pluggable storage backends, making it suitable for both single-instance and distributed applications.

## Features

- Five rate limiting algorithms out of the box
- Pluggable storage backends (in-memory, Redis, PostgreSQL)
- Context-aware — cancellation and deadlines propagate through all operations
- Thread-safe with mutex-based synchronization
- Common interface across all algorithms for easy swapping
- Sub-millisecond performance on most algorithms
- Minimal memory allocations

## Used In Production

- [TodoAPI](https://todoflow-black.vercel.app) - Production Todo API demonstrating Redis → PostgreSQL failover strategy with distributed rate limiting
- Add yours via PR!

## Installation

```bash
go get github.com/codetesla51/limitz
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/codetesla51/limitz/algorithms"
    "github.com/codetesla51/limitz/store"
)

func main() {
    s := store.NewMemoryStore()
    defer s.Close()

    limiter := algorithms.NewTokenBucket(10, 5, s)

    ctx := context.Background()
    result, err := limiter.Allow(ctx, "user-123")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    if result.Allowed {
        fmt.Printf("Request allowed. Remaining: %d\n", result.Remaining)
    } else {
        fmt.Printf("Rate limited. Retry after: %v\n", result.RetryAfter)
    }
}
```

## Interface

All algorithms implement the `RateLimiter` interface:

```go
type RateLimiter interface {
    Allow(ctx context.Context, key string) (Result, error)
    Reset(ctx context.Context, key string) error
}
```

Each call to `Allow` returns a `Result`:

```go
type Result struct {
    Allowed    bool
    Limit      int
    Remaining  int
    RetryAfter time.Duration
}
```

### Context Support

Every method accepts a `context.Context`. This means:

- If the caller's context is cancelled or times out, the operation aborts immediately
- In HTTP handlers, pass `r.Context()` directly — it cancels automatically if the client disconnects
- You can set per-operation timeouts using `context.WithTimeout`

```go
// In an HTTP handler — context cancels if client disconnects
result, err := limiter.Allow(r.Context(), r.RemoteAddr)

// With an explicit timeout on the rate limit check
ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
defer cancel()
result, err := limiter.Allow(ctx, r.RemoteAddr)
```

---

## Algorithms

### Token Bucket

Tokens are added to a bucket at a fixed rate. Each request consumes one token. Requests are denied when the bucket is empty. Allows short bursts up to the bucket capacity.

```go
// capacity: max tokens the bucket can hold
// refillRate: tokens added per second
limiter := algorithms.NewTokenBucket(100, 10, s)
```

Best for: APIs that need to allow short bursts while enforcing an average rate.

---

### Fixed Window

Divides time into fixed-duration windows and counts requests within each window. The counter resets at the start of each new window.

```go
// limit: max requests per window
// windowSize: duration of each window
limiter := algorithms.NewFixedWindow(100, 1*time.Minute, s)
```

Best for: Simple rate limiting where boundary precision is not critical.

Note: Susceptible to burst traffic at window boundaries. A client could send the maximum number of requests at the end of one window and again at the start of the next, effectively doubling throughput in a short period.

---

### Sliding Window (Log)

Tracks the exact timestamp of every request. Filters out timestamps outside the current window on each request. Provides the most accurate rate limiting.

```go
// limit: max requests per window
// windowSize: sliding window duration
limiter := algorithms.NewSlidingWindow(100, 1*time.Minute, s)
```

Best for: Applications that require precise rate limiting with no boundary issues.

Trade-off: Higher memory usage and slower performance due to storing individual timestamps. See benchmarks below.

---

### Sliding Window Counter

A hybrid approach that approximates the sliding window using two fixed windows. Tracks request counts for the current and previous window, then estimates the request rate based on how far into the current window the request falls.

```go
// limit: max requests per window
// windowSize: window duration
limiter := algorithms.NewSlidingWindowCounter(100, 1*time.Minute, s)
```

Best for: Applications that need sliding window accuracy without the memory overhead of the full sliding window log.

---

### Leaky Bucket

Models a bucket with a fixed-size queue that leaks (processes) requests at a constant rate. Incoming requests are added to the queue. If the queue is full, requests are rejected.

```go
// capacity: max queue size
// rate: requests processed (leaked) per second
limiter := algorithms.NewLeakyBucket(100, 10, s)
```

Best for: Smoothing out bursty traffic into a steady stream.

---

## Algorithm Comparison

| Algorithm              | Burst Handling | Memory Usage | Accuracy    | Boundary Issues |
|------------------------|----------------|--------------|-------------|-----------------|
| Token Bucket           | Allows bursts  | Low          | Good        | None            |
| Fixed Window           | Allows bursts  | Low          | Moderate    | Yes             |
| Sliding Window (Log)   | No bursts      | High         | Exact       | None            |
| Sliding Window Counter | Limited        | Low          | Approximate | Minimal         |
| Leaky Bucket           | No bursts      | Low          | Good        | None            |

---

## Storage Backends

All storage backends implement the `Store` interface:

```go
type Store interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

### In-Memory

Default storage backend. Data is held in a Go map with automatic expiration cleanup running in the background. Suitable for single-instance applications.

```go
s := store.NewMemoryStore()
defer s.Close()
```

- No external dependencies
- Fastest performance
- Data is lost on restart
- Not shared across instances

### Redis

Distributed storage backend using Redis. Supports authentication, connection pooling, and automatic retries. Suitable for multi-instance deployments.

```go
s, err := store.NewRedisStore("localhost:6379", "username", "password")
if err != nil {
    log.Fatal(err)
}
defer s.Close()
```

The Redis store uses the following connection settings:

```go
client := redis.NewClient(&redis.Options{
    Addr:         addr,
    Username:     username,
    Password:     password,
    MaxRetries:   3,
    PoolSize:     10,
    MinIdleConns: 5,
})
```

- Shared state across instances
- Persistent across restarts
- Requires a running Redis server
- Pass empty strings for username/password if authentication is not configured
- Automatic connection retry (up to 3 attempts)
- The caller's context is passed directly to every Redis operation — if the context times out or is cancelled, the Redis call aborts

### PostgreSQL

Persistent storage backend using PostgreSQL via GORM. Tables and indexes are created automatically on initialization.

```go
dsn := "host=localhost user=postgres password=secret dbname=ratelimit port=5432"
s, err := store.NewDatabaseStore(dsn)
if err != nil {
    log.Fatal(err)
}
defer s.Close()
```

- Durable, persistent storage
- Leverages existing database infrastructure
- Higher latency compared to in-memory and Redis
- The caller's context is passed to every query via `db.WithContext(ctx)`
- Call `s.CleanupExpired()` periodically to remove stale entries

---

## HTTP Middleware Example

Limitz has no HTTP dependencies. Here is an example middleware for `net/http`:

```go
func RateLimitMiddleware(limiter algorithms.RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            result, err := limiter.Allow(r.Context(), r.RemoteAddr)
            if err != nil {
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
            w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))

            if !result.Allowed {
                w.Header().Set("Retry-After", fmt.Sprintf("%d", int(result.RetryAfter.Seconds())))
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

Usage:

```go
s := store.NewMemoryStore()
limiter := algorithms.NewTokenBucket(100, 10, s)
mux := http.NewServeMux()
mux.HandleFunc("/", handler)
http.ListenAndServe(":8080", RateLimitMiddleware(limiter)(mux))
```

---

## Examples

### Basic Rate Limiting

```go
s := store.NewMemoryStore()
defer s.Close()

ctx := context.Background()
limiter := algorithms.NewFixedWindow(5, 1*time.Second, s)

for i := 0; i < 8; i++ {
    result, _ := limiter.Allow(ctx, "user-1")
    fmt.Printf("Request %d: allowed=%v remaining=%d\n", i+1, result.Allowed, result.Remaining)
}
```

Output:

```
Request 1: allowed=true remaining=4
Request 2: allowed=true remaining=3
Request 3: allowed=true remaining=2
Request 4: allowed=true remaining=1
Request 5: allowed=true remaining=0
Request 6: allowed=false remaining=0
Request 7: allowed=false remaining=0
Request 8: allowed=false remaining=0
```

### Per-User Rate Limiting

Each key gets its own independent rate limit:

```go
s := store.NewMemoryStore()
defer s.Close()

ctx := context.Background()
limiter := algorithms.NewTokenBucket(5, 2, s)

limiter.Allow(ctx, "alice") // allowed, alice has 4 remaining
limiter.Allow(ctx, "alice") // allowed, alice has 3 remaining
limiter.Allow(ctx, "bob")   // allowed, bob has 4 remaining (independent)
```

### Per-Endpoint Rate Limiting

Use composite keys to rate limit per user per endpoint:

```go
ctx := context.Background()
s := store.NewMemoryStore()
defer s.Close()

writeLimiter := algorithms.NewSlidingWindowCounter(10, 1*time.Minute, s)
readLimiter := algorithms.NewSlidingWindowCounter(100, 1*time.Minute, s)

userID := "user-42"
writeLimiter.Allow(ctx, userID+":/api/posts")
readLimiter.Allow(ctx, userID+":/api/feed")
```

### Handling Rate Limit Results

```go
ctx := context.Background()
s := store.NewMemoryStore()
defer s.Close()

limiter := algorithms.NewLeakyBucket(3, 1, s)

for i := 0; i < 5; i++ {
    result, err := limiter.Allow(ctx, "client-1")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        continue
    }

    if result.Allowed {
        fmt.Printf("Request %d: allowed (remaining: %d)\n", i+1, result.Remaining)
    } else {
        fmt.Printf("Request %d: denied (retry after: %v)\n", i+1, result.RetryAfter)
    }
}
```

### Resetting a Rate Limit

```go
ctx := context.Background()
s := store.NewMemoryStore()
defer s.Close()

limiter := algorithms.NewTokenBucket(5, 1, s)

for i := 0; i < 5; i++ {
    limiter.Allow(ctx, "user-1")
}

result, _ := limiter.Allow(ctx, "user-1")
fmt.Println(result.Allowed) // false

limiter.Reset(ctx, "user-1")

result, _ = limiter.Allow(ctx, "user-1")
fmt.Println(result.Allowed) // true
```

### Swapping Algorithms

All algorithms share the same interface:

```go
ctx := context.Background()
s := store.NewMemoryStore()
defer s.Close()

var limiter algorithms.RateLimiter

limiter = algorithms.NewTokenBucket(100, 10, s)
// limiter = algorithms.NewFixedWindow(100, 1*time.Minute, s)
// limiter = algorithms.NewLeakyBucket(100, 10, s)
// limiter = algorithms.NewSlidingWindow(100, 1*time.Minute, s)
// limiter = algorithms.NewSlidingWindowCounter(100, 1*time.Minute, s)

result, _ := limiter.Allow(ctx, "user-1")
fmt.Println(result.Allowed)
```

### Distributed Rate Limiting with Redis

```go
ctx := context.Background()
s, err := store.NewRedisStore("localhost:6379", "", "")
if err != nil {
    log.Fatal(err)
}
defer s.Close()

limiter := algorithms.NewSlidingWindowCounter(1000, 1*time.Minute, s)

result, _ := limiter.Allow(ctx, "api-key-xyz")
fmt.Printf("Allowed: %v, Remaining: %d\n", result.Allowed, result.Remaining)
```

---

## Benchmarks

Benchmarks were run on an Intel Core i5-6300U @ 2.40GHz, 4 threads, Linux/amd64.

### Single User (Sequential)

| Algorithm              | ops/sec   | ns/op | B/op  | allocs/op |
|------------------------|-----------|-------|-------|-----------|
| Token Bucket           | 885,632   | 1,180 | 48    | 1         |
| Fixed Window           | 861,370   | 1,217 | 48    | 1         |
| Leaky Bucket           | 1,096,038 | 1,234 | 48    | 1         |
| Sliding Window Counter | 980,431   | 1,361 | 48    | 1         |
| Sliding Window (Log)   | 216,994   | 6,483 | 2,087 | 8         |

### Multiple Users (Sequential)

| Algorithm              | ops/sec | ns/op | B/op  | allocs/op |
|------------------------|---------|-------|-------|-----------|
| Token Bucket           | 795,757 | 1,542 | 55    | 2         |
| Fixed Window           | 751,354 | 1,464 | 55    | 2         |
| Leaky Bucket           | 713,905 | 1,528 | 55    | 2         |
| Sliding Window Counter | 795,951 | 1,413 | 55    | 2         |
| Sliding Window (Log)   | 500,776 | 6,316 | 1,921 | 9         |

### Concurrent (Parallel)

| Algorithm              | ops/sec | ns/op   | B/op    | allocs/op |
|------------------------|---------|---------|---------|-----------|
| Token Bucket           | 604,676 | 1,749   | 48      | 1         |
| Fixed Window           | 653,539 | 1,949   | 48      | 1         |
| Leaky Bucket           | 666,572 | 2,010   | 48      | 1         |
| Sliding Window Counter | 592,909 | 1,996   | 48      | 1         |
| Sliding Window (Log)   | 10,000  | 178,213 | 108,436 | 16        |

Token Bucket, Fixed Window, Leaky Bucket, and Sliding Window Counter all perform within the same range at roughly 1,200–2,000 ns/op with minimal allocations. Sliding Window (Log) is significantly slower due to the overhead of storing and filtering individual timestamps, and is not recommended for high-throughput concurrent workloads.

Run benchmarks locally:

```bash
go test ./algorithms -bench=. -benchmem
```

---

## Testing

```bash
go test ./algorithms -v
```

---

## License

See [LICENSE](LICENSE) for details.
