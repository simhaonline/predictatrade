---
name: go-concurrency-patterns
description: "Go concurrency patterns for tick processing and pipelines."
---

# go-concurrency-patterns

Use when writing Go concurrency code for Predict-A-Trade realtime engine: tick processing pipelines, fan-out/fan-in, worker pools, or signal delivery.

## Project-Specific Patterns

### Tick Processing Pipeline
```go
ctx, cancel := context.WithCancel(parentCtx)
defer cancel()
ticks := make(chan *Tick, 4096)     // buffered for burst
features := make(chan *Feature, 256)
signals := make(chan *Signal, 64)
```

### Worker Pool (for signal broadcast)
```go
const workers = 16
for i := 0; i < workers; i++ {
    go signalWorker(ctx, signals, clients)
}
```

### Graceful Shutdown
```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh
cancel()
// drain channels, flush buffers, close connections
```

### Context Propagation
All external calls must carry ctx: pgx (ctx), redis (ctx), HTTP (req.WithContext(ctx))

## Pitfalls
- Unbuffered channel between fast producer and slow consumer = producer blocked
- Forgetting defer cancel() leaks goroutines
- range over channel never exits if producer doesn't close
- sync.Mutex held during I/O = whole pipeline stalls
