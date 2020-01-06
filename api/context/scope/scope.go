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

// TODO:
// (6) Add static Meter.New*() helpers, give (ctx context.Context) param to all Meter.New* APIs.
// (9) Get first-class about Namespaces

package scope

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/api/context/baggage"
	"go.opentelemetry.io/otel/api/context/label"
	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

type (
	Scope struct {
		*scopeImpl
	}

	scopeImpl struct {
		resources   baggage.Map
		provider    *Provider
		scopeTracer scopeTracer
		scopeMeter  scopeMeter
	}

	scopeTracer struct {
		*scopeImpl
	}

	scopeMeter struct {
		*scopeImpl
	}

	Provider struct {
		tracer      trace.Tracer
		meter       metric.Meter
		propagators propagation.Propagators
	}
)

const (
	namespaceKey core.Key = "$namespace"
)

var (
	_ trace.Tracer = &scopeTracer{}
	_ metric.Meter = &scopeMeter{}

	nilProvider = &Provider{}
)

func NewProvider(t trace.Tracer, m metric.Meter, p propagation.Propagators) *Provider {
	return &Provider{
		tracer:      t,
		meter:       m,
		propagators: p,
	}
}

func (p *Provider) Tracer() trace.Tracer {
	return p.tracer
}

func (p *Provider) Meter() metric.Meter {
	return p.meter
}

func (p *Provider) Propagators() propagation.Propagators {
	return p.propagators
}

func (p *Provider) New() Scope {
	si := &scopeImpl{
		resources: baggage.Empty(),
		provider:  p,
	}
	si.scopeMeter.scopeImpl = si
	si.scopeTracer.scopeImpl = si
	return Scope{si}
}

func Empty() Scope {
	return Scope{}
}

// func (si *scopeImpl) copy(s Scope) {
// 	si.provider = nilProvider
// 	if s.scopeImpl != nil {
// 		si = *s.scopeImpl
// 	}
// 	si.scopeMeter.scopeImpl = &si
// 	si.scopeTracer.scopeImpl = &si
// }

func (s Scope) clone() Scope {
	var ri scopeImpl
	if s.scopeImpl != nil {
		ri.provider = s.provider
		ri.resources = s.resources
	} else {
		ri.provider = nilProvider
	}
	ri.scopeMeter.scopeImpl = &ri
	ri.scopeTracer.scopeImpl = &ri
	return Scope{
		scopeImpl: &ri,
	}
}

func (s Scope) Named(name string) Scope {
	r := s.clone()
	r.resources = s.resources.Apply(baggage.MapUpdate{
		SingleKV: namespaceKey.String(name),
	})
	return r
}

func (s Scope) WithMeter(meter metric.Meter) Scope {
	r := s.clone()
	r.provider = NewProvider(r.provider.tracer, meter, r.provider.propagators)
	return r
}

func (s Scope) WithTracer(tracer trace.Tracer) Scope {
	r := s.clone()
	r.provider = NewProvider(tracer, r.provider.meter, r.provider.propagators)
	return r
}

func (s Scope) WithPropagators(propagators propagation.Propagators) Scope {
	r := s.clone()
	r.provider = NewProvider(r.provider.tracer, r.provider.meter, propagators)
	return r
}

func (s Scope) Provider() *Provider {
	if s.scopeImpl == nil {
		return nilProvider
	}
	return s.provider
}

func (s Scope) Resources() baggage.Map {
	if s.scopeImpl == nil {
		return baggage.Empty()
	}
	return s.resources
}

func (s Scope) Name() string {
	val, _ := s.Resources().Value(namespaceKey)
	return val.AsString()
}

func (s Scope) Tracer() trace.Tracer {
	if s.scopeImpl == nil {
		return trace.NoopTracer{}
	}
	return &s.scopeTracer
}

func (s Scope) Meter() metric.Meter {
	if s.scopeImpl == nil {
		return metric.NoopMeter{}
	}
	return &s.scopeMeter
}

func (s Scope) Propagators() propagation.Propagators {
	if s.scopeImpl == nil {
		// TODO
		// return propagation.Default
	}
	return s.provider.Propagators()
}

func (s *scopeImpl) enterScope(ctx context.Context) context.Context {
	o := Current(ctx)
	if o.scopeImpl == s {
		return ctx
	}
	return ContextWithScope(ctx, Scope{s})
}

func (s *scopeImpl) subname(name string) string {
	ns, _ := s.resources.Value(namespaceKey)
	str := ns.AsString()
	if str == "" {
		return name
	}
	return str + "/" + name
}

func (s *scopeTracer) Start(
	ctx context.Context,
	name string,
	opts ...trace.StartOption,
) (context.Context, trace.Span) {
	if s.scopeImpl == nil {
		return ctx, trace.NoopSpan{}
	}
	return s.provider.Tracer().Start(s.enterScope(ctx), s.subname(name), opts...)
}

func (s *scopeTracer) WithSpan(
	ctx context.Context,
	name string,
	fn func(ctx context.Context) error,
) error {
	if s.scopeImpl == nil {
		return fn(ctx)
	}
	return s.provider.Tracer().WithSpan(s.enterScope(ctx), s.subname(name), fn)
}

func (m *scopeMeter) NewInt64Counter(name string, cos ...metric.CounterOptionApplier) metric.Int64Counter {
	return m.provider.Meter().NewInt64Counter(m.subname(name), cos...)
}

func (m *scopeMeter) NewFloat64Counter(name string, cos ...metric.CounterOptionApplier) metric.Float64Counter {
	return m.provider.Meter().NewFloat64Counter(m.subname(name), cos...)
}

func (m *scopeMeter) NewInt64Gauge(name string, gos ...metric.GaugeOptionApplier) metric.Int64Gauge {
	return m.provider.Meter().NewInt64Gauge(m.subname(name), gos...)
}

func (m *scopeMeter) NewFloat64Gauge(name string, gos ...metric.GaugeOptionApplier) metric.Float64Gauge {
	return m.provider.Meter().NewFloat64Gauge(m.subname(name), gos...)
}

func (m *scopeMeter) NewInt64Measure(name string, mos ...metric.MeasureOptionApplier) metric.Int64Measure {
	return m.provider.Meter().NewInt64Measure(m.subname(name), mos...)
}

func (m *scopeMeter) NewFloat64Measure(name string, mos ...metric.MeasureOptionApplier) metric.Float64Measure {
	return m.provider.Meter().NewFloat64Measure(m.subname(name), mos...)
}

func (m *scopeMeter) RecordBatch(ctx context.Context, labels label.Set, ms ...metric.Measurement) {
	m.provider.Meter().RecordBatch(m.enterScope(ctx), labels, ms...)
}

func (s Scope) String() string {
	if s.scopeImpl == nil {
		return "{ empty }"
	}
	return fmt.Sprintf("{ res=%v }", s.resources)
}
