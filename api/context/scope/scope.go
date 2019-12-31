// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// SO.. TODO:
// (1) Let the Tracer() and Meter() stubs returned here set the current scope and call through
// (2) Place a TODO about support for scope-ful Propagators.
// (3) Library name/version are in resources
// (4) Resources are LabelSets
// (5) Label API moves into api/label
// (6) Add static Meter.New*() helpers, give (ctx context.Context) param to all Meter.New* APIs.
// (7) Move current Span/SpanContext into a current Scope method, a new With() scope caller,
// (8) scope.Provider TODO below on inerface is needed.  global
//     forwarder and scope forwarder should just implement the 9 methods.

package scope

import (
	"context"

	"go.opentelemetry.io/otel/api/context/baggage"
	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

type (
	Scope struct {
		*scopeImpl
	}

	scopeImpl struct {
		resources baggage.Map
		provider  Provider
		scopeTracer
		scopeMeter
	}

	Provider interface {
		// TODO is this interface needed?  Only the global uses it, seems not needed.

		Tracer() trace.Tracer
		Meter() metric.Meter
		Propagators() propagation.Propagators
	}

	scopeTracer struct {
		*scopeImpl
	}

	scopeMeter struct {
		*scopeImpl
	}

	provider struct {
		tracer      trace.Tracer
		meter       metric.Meter
		propagators propagation.Propagators
	}
)

var _ trace.Tracer = &scopeTracer{}

//var _ metric.Meter = &scopeMeter{}

func NewProvider(t trace.Tracer, m metric.Meter, p propagation.Propagators) Provider {
	return &provider{
		tracer:      t,
		meter:       m,
		propagators: p,
	}
}

func New(resources baggage.Map, provider Provider) Scope {
	si := &scopeImpl{
		resources: resources,
		provider:  provider,
	}
	si.scopeMeter.scopeImpl = si
	si.scopeTracer.scopeImpl = si
	return Scope{
		scopeImpl: si,
	}
}

func (p *provider) Tracer() trace.Tracer {
	return p.tracer
}

func (p *provider) Meter() metric.Meter {
	return p.meter
}

func (p *provider) Propagators() propagation.Propagators {
	return p.propagators
}

func (s Scope) Resources() baggage.Map {
	return s.resources
}

func (s Scope) Tracer() trace.Tracer {
	return &s.scopeTracer
}

func (s Scope) Meter() metric.Meter {
	// return &s.scopeMeter
	// @@@
	return s.provider.Meter()
}

func (s Scope) Propagators() propagation.Propagators {
	return s.provider.Propagators()
}

func (s *scopeImpl) enterScope(ctx context.Context) context.Context {
	return ctx
}

func (s *scopeTracer) Start(
	ctx context.Context,
	spanName string,
	opts ...trace.StartOption,
) (context.Context, trace.Span) {
	return s.provider.Tracer().Start(s.enterScope(ctx), spanName, opts...)
}

func (s *scopeTracer) WithSpan(
	ctx context.Context,
	spanName string,
	fn func(ctx context.Context) error,
) error {
	return s.provider.Tracer().WithSpan(s.enterScope(ctx), spanName, fn)
}
