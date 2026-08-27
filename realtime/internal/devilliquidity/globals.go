package devilliquidity

import "sync"

// Global engine accessor so the gateway HTTP server (which has a fixed
// constructor signature) can expose Devil Liquidity state without coupling.
var (
	globalMu     sync.RWMutex
	globalEngine *Engine
)

// SetGlobalEngine registers the process-wide engine instance.
func SetGlobalEngine(e *Engine) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalEngine = e
}

// GlobalEngine returns the registered engine, or nil.
func GlobalEngine() *Engine {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalEngine
}
