---
name: go-race-profiling
description: "Profile Go for races, leaks, latency, and allocations."
---

# go-race-profiling

Use when profiling Predict-A-Trade Go realtime engine for performance, race conditions, goroutine leaks, or allocation pressure.

## Commands
```bash
cd realtime
go test -race -count=1 ./...                          # race detection
go test -bench=. -benchmem -benchtime=10s ./...       # benchmarks + allocs
go test -cpuprofile=cpu.prof -memprofile=mem.prof     # profiling
go tool pprof -http=:6060 cpu.prof                    # visualize
go test -blockprofile=block.prof                      # blocking ops
go vet ./...                                          # static analysis
```

## Key Targets
- Gate pipeline: < 1ms per candidate (p99)
- Signal compute: < 50ms end-to-end (p99)
- WS delivery: < 10ms per broadcast
- Goroutines: no leaks over 24h run (go test -race finds leaks)

## Patterns to Audit
- Every go func() must accept context.Context for cancellation
- Unbuffered channels in hot path = deadlock risk
- sync.Pool for hot allocations (tick buffers, JSON encoders)
- atomic operations for counters shared across goroutines
- time.Ticker.Stop() called in defer
- context.WithTimeout on all external calls (DB, Redis, HTTP)
