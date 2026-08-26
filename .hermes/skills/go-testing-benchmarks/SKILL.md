---
name: go-testing-benchmarks
description: "Go test patterns, golden fixtures, and table-driven tests."
---

# go-testing-benchmarks

Use when writing Go tests for Predict-A-Trade: unit tests, golden/parity fixtures, benchmarks, or integration tests.

## Test Patterns in Use

### Table-Driven Tests
```go
func TestStrategySignal(t *testing.T) {
    tests := []struct{name string; input Candle; want Signal}{...}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := engine.Evaluate(tt.input)
            if got.Direction != tt.want.Direction { t.Errorf(...) }
        })
    }
}
```

### Golden Fixture Tests
Files: realtime/internal/strategy/golden_test.go
Run: go test -update        (regenerate golden files)
Golden dir: realtime/testdata/golden/

### Benchmark Tests
Run: go test -bench=. -benchtime=30s -count=5
Key benchmarks: Strategy evaluation, Gate pipeline, Signal delivery

### Race Detector
Run: go test -race -count=5 ./internal/strategy/
Race bugs found in signal engine, WS broadcast, cache access — always run before PR.

## Build Verification
```bash
cd realtime
go test -race ./...          # 30/30 packages, 0 failures
go vet ./...
go build ./cmd/...
```
