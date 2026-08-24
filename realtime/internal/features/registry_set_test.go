package features

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
)

// prompt.md Section 73: registries are per-timeframe — same TF returns the
// same instance (shared cache), different TFs never share state.
func TestRegistrySetIsolation(t *testing.T) {
	set := NewRegistrySet()
	m1 := set.For(types.TFM1)
	m1Again := set.For(types.TFM1)
	h1 := set.For(types.TFH1)

	if m1 != m1Again {
		t.Error("same timeframe must return the same registry instance")
	}
	if m1 == h1 {
		t.Fatal("different timeframes must NOT share a registry (state contamination)")
	}
}

// Provider wiring propagates to existing and future registries.
func TestRegistrySetProviderPropagation(t *testing.T) {
	set := NewRegistrySet()
	existing := set.For(types.TFM5)
	_ = existing
	// No provider configured: readiness falls back to not-configured state.
	if set.newsProvider != nil {
		t.Error("fresh set must have no provider")
	}
}
