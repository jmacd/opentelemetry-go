package internal

import (
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/context/scope"
	tracePropagation "go.opentelemetry.io/otel/api/trace/propagation"
)

type (
	scopeHolder struct {
		sc scope.Scope
	}
)

var (
	globalScope  = defaultScopeValue()
	delegateOnce sync.Once
)

// Scope is the internal implementation for global.Scope().
func Scope() scope.Scope {
	return globalScope.Load().(scopeHolder).sc
}

// SetScope is the internal implementation for global.SetScope().
func SetScope(sc scope.Scope) {
	first := false
	delegateOnce.Do(func() {
		current := Scope()
		currentProvider := current.Provider()
		newProvider := sc.Provider()

		first = true

		if currentProvider.Meter() == newProvider.Meter() {
			// Setting the global scope to former default is nonsense, panic.
			// Panic is acceptable because we are likely still early in the
			// process lifetime.
			panic("invalid Provider, the global instance cannot be reinstalled")
		} else if deft, ok := currentProvider.Tracer().(*tracer); ok {
			deft.deferred.setDelegate(sc)
		} else {
			panic("impossible error")
		}
	})
	if !first {
		panic("global scope has already been initialized")
	}
	globalScope.Store(scopeHolder{sc: sc})
}

func defaultScopeValue() *atomic.Value {
	v := &atomic.Value{}
	d := newDeferred()
	v.Store(scopeHolder{
		sc: scope.NewProvider(&d.tracer, &d.meter, &d.propagators).New(),
	})
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
