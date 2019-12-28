package internal

import (
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/context/scope"
	tracePropagation "go.opentelemetry.io/otel/api/trace/propagation"
)

type (
	scopeProviderHolder struct {
		sp scope.Provider
	}
)

var (
	globalScope  = defaultScopeValue()
	delegateOnce sync.Once
)

// ScopeProvider is the internal implementation for global.ScopeProvider.
func ScopeProvider() scope.Provider {
	return globalScope.Load().(scopeProviderHolder).sp
}

// SetScopeProvider is the internal implementation for global.SetScopeProvider.
func SetScopeProvider(sp scope.Provider) {
	delegateOnce.Do(func() {
		current := ScopeProvider()

		if current == sp {
			// Setting the provider to the prior default is nonsense, panic.
			// Panic is acceptable because we are likely still early in the
			// process lifetime.
			panic("invalid Provider, the global instance cannot be reinstalled")
		} else if def, ok := current.(*deferred); ok {
			def.setDelegate(sp)
		}
	})
	globalScope.Store(scopeProviderHolder{sp: sp})
}

func defaultScopeValue() *atomic.Value {
	v := &atomic.Value{}
	v.Store(scopeProviderHolder{sp: newDeferred()})
	return v
}

// getDefaultPropagators returns a default Propagators, configured
// with W3C trace context propagation.
func getDefaultPropagators() propagation.Propagators {
	inex := tracePropagation.TraceContext{}
	return propagation.New(
		propagation.WithExtractors(inex),
		propagation.WithInjectors(inex),
	)
}

// ResetForTest restores the initial global state, for testing purposes.
func ResetForTest() {
	globalScope = defaultScopeValue()
	delegateOnce = sync.Once{}
}
