package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"

	"go.opentelemetry.io/otel/api/context/baggage"
	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/context/scope"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

// This file contains the forwarding implementation of metric.Provider
// used as the default global instance.  Metric events using instruments
// provided by this implementation are no-ops until the first Meter
// implementation is set as the global provider.
//
// The implementation here uses Mutexes to maintain a list of active
// Meters in the Provider and Instruments in each Meter, under the
// assumption that these interfaces are not performance-critical.
//
// We have the invariant that setDelegate() will be called before a
// new metric.Provider implementation is registered as the global
// provider.  Mutexes in the Provider and Meters ensure that each
// instrument has a delegate before the global provider is set.
//
// LabelSets are implemented by delegating to the Meter instance using
// the metric.LabelSetDelegator interface.
//
// Bound instrument operations are implemented by delegating to the
// instrument after it is registered, with a sync.Once initializer to
// protect against races with Release().

type metricKind int8

const (
	counterKind metricKind = iota
	gaugeKind
	measureKind
)

type deferred struct {
	lock        sync.Mutex
	meter       meter
	tracer      tracer
	propagators propagators
	resources   baggage.Map
	delegate    unsafe.Pointer // (*scope.Scope)
}

type meter struct {
	deferred *deferred

	instruments []*instImpl
}

type tracer struct {
	deferred *deferred

	trace.NoopTracer
}

type propagators struct {
	deferred *deferred

	propagation.Propagators
}

type instImpl struct {
	meter *meter

	name  string
	mkind metricKind
	nkind core.NumberKind
	opts  interface{}

	delegate unsafe.Pointer // (*metric.InstrumentImpl)
}

type instBound struct {
	inst   *instImpl
	labels core.LabelSet

	initialize sync.Once
	delegate   unsafe.Pointer // (*metric.BoundImpl)
}

var _ metric.Meter = &meter{}
var _ metric.InstrumentImpl = &instImpl{}
var _ metric.BoundInstrumentImpl = &instBound{}

// Provider interface and delegation

func newDeferred() *deferred {
	d := &deferred{}
	d.meter.deferred = d
	d.tracer.deferred = d
	d.propagators.deferred = d
	d.propagators.Propagators = getDefaultPropagators()
	return d
}

func (d *deferred) setDelegate(sc scope.Scope) {
	d.lock.Lock()
	defer d.lock.Unlock()

	ptr := unsafe.Pointer(&sc)
	atomic.StorePointer(&d.delegate, ptr)

	d.meter.setDelegate(sc)
}

func (d *deferred) Tracer() trace.Tracer {
	if implPtr := (*scope.Provider)(atomic.LoadPointer(&d.delegate)); implPtr != nil {
		return (*implPtr).Tracer()
	}
	return &d.tracer
}

func (d *deferred) Meter() metric.Meter {
	if implPtr := (*scope.Provider)(atomic.LoadPointer(&d.delegate)); implPtr != nil {
		return (*implPtr).Meter()
	}
	return &d.meter
}

func (d *deferred) Propagators() propagation.Propagators {
	if implPtr := (*scope.Provider)(atomic.LoadPointer(&d.delegate)); implPtr != nil {
		return (*implPtr).Propagators()
	}
	return &d.propagators
}

// Meter interface

func (m *meter) setDelegate(sc scope.Scope) {
	for _, i := range m.instruments {
		i.setDelegate(sc)
	}
	m.instruments = nil
}

func (m *meter) newInst(name string, mkind metricKind, nkind core.NumberKind, opts interface{}) metric.InstrumentImpl {
	m.deferred.lock.Lock()
	defer m.deferred.lock.Unlock()

	if implPtr := (*scope.Provider)(atomic.LoadPointer(&m.deferred.delegate)); implPtr != nil {
		return newInstDelegate((*implPtr).Meter(), name, mkind, nkind, opts)
	}

	inst := &instImpl{
		meter: m,
		name:  name,
		mkind: mkind,
		nkind: nkind,
		opts:  opts,
	}
	m.instruments = append(m.instruments, inst)
	return inst
}

func newInstDelegate(m metric.Meter, name string, mkind metricKind, nkind core.NumberKind, opts interface{}) metric.InstrumentImpl {
	switch mkind {
	case counterKind:
		if nkind == core.Int64NumberKind {
			return m.NewInt64Counter(name, opts.([]metric.CounterOptionApplier)...).Impl()
		}
		return m.NewFloat64Counter(name, opts.([]metric.CounterOptionApplier)...).Impl()
	case gaugeKind:
		if nkind == core.Int64NumberKind {
			return m.NewInt64Gauge(name, opts.([]metric.GaugeOptionApplier)...).Impl()
		}
		return m.NewFloat64Gauge(name, opts.([]metric.GaugeOptionApplier)...).Impl()
	case measureKind:
		if nkind == core.Int64NumberKind {
			return m.NewInt64Measure(name, opts.([]metric.MeasureOptionApplier)...).Impl()
		}
		return m.NewFloat64Measure(name, opts.([]metric.MeasureOptionApplier)...).Impl()
	}
	return nil
}

// Instrument delegation

func (inst *instImpl) setDelegate(provider scope.Scope) {
	implPtr := new(metric.InstrumentImpl)

	*implPtr = newInstDelegate(provider.Meter(), inst.name, inst.mkind, inst.nkind, inst.opts)

	atomic.StorePointer(&inst.delegate, unsafe.Pointer(implPtr))
}

func (inst *instImpl) Bind(labels core.LabelSet) metric.BoundInstrumentImpl {
	if implPtr := (*metric.InstrumentImpl)(atomic.LoadPointer(&inst.delegate)); implPtr != nil {
		return (*implPtr).Bind(labels)
	}
	return &instBound{
		inst:   inst,
		labels: labels,
	}
}

func (bound *instBound) Unbind() {
	bound.initialize.Do(func() {})

	implPtr := (*metric.BoundInstrumentImpl)(atomic.LoadPointer(&bound.delegate))

	if implPtr == nil {
		return
	}

	(*implPtr).Unbind()
}

// Metric updates

func (m *meter) RecordBatch(ctx context.Context, labels core.LabelSet, measurements ...metric.Measurement) {
	if delegatePtr := (*scope.Provider)(atomic.LoadPointer(&m.deferred.delegate)); delegatePtr != nil {
		(*delegatePtr).Meter().RecordBatch(ctx, labels, measurements...)
	}
}

func (inst *instImpl) RecordOne(ctx context.Context, number core.Number, labels core.LabelSet) {
	if instPtr := (*metric.InstrumentImpl)(atomic.LoadPointer(&inst.delegate)); instPtr != nil {
		(*instPtr).RecordOne(ctx, number, labels)
	}
}

// Bound instrument initialization

func (bound *instBound) RecordOne(ctx context.Context, number core.Number) {
	instPtr := (*metric.InstrumentImpl)(atomic.LoadPointer(&bound.inst.delegate))
	if instPtr == nil {
		return
	}
	var implPtr *metric.BoundInstrumentImpl
	bound.initialize.Do(func() {
		implPtr = new(metric.BoundInstrumentImpl)
		*implPtr = (*instPtr).Bind(bound.labels)
		atomic.StorePointer(&bound.delegate, unsafe.Pointer(implPtr))
	})
	if implPtr == nil {
		implPtr = (*metric.BoundInstrumentImpl)(atomic.LoadPointer(&bound.delegate))
	}
	(*implPtr).RecordOne(ctx, number)
}

// Constructors

func (m *meter) NewInt64Counter(name string, opts ...metric.CounterOptionApplier) metric.Int64Counter {
	return metric.WrapInt64CounterInstrument(m.newInst(name, counterKind, core.Int64NumberKind, opts))
}

func (m *meter) NewFloat64Counter(name string, opts ...metric.CounterOptionApplier) metric.Float64Counter {
	return metric.WrapFloat64CounterInstrument(m.newInst(name, counterKind, core.Float64NumberKind, opts))
}

func (m *meter) NewInt64Gauge(name string, opts ...metric.GaugeOptionApplier) metric.Int64Gauge {
	return metric.WrapInt64GaugeInstrument(m.newInst(name, gaugeKind, core.Int64NumberKind, opts))
}

func (m *meter) NewFloat64Gauge(name string, opts ...metric.GaugeOptionApplier) metric.Float64Gauge {
	return metric.WrapFloat64GaugeInstrument(m.newInst(name, gaugeKind, core.Float64NumberKind, opts))
}

func (m *meter) NewInt64Measure(name string, opts ...metric.MeasureOptionApplier) metric.Int64Measure {
	return metric.WrapInt64MeasureInstrument(m.newInst(name, measureKind, core.Int64NumberKind, opts))
}

func (m *meter) NewFloat64Measure(name string, opts ...metric.MeasureOptionApplier) metric.Float64Measure {
	return metric.WrapFloat64MeasureInstrument(m.newInst(name, measureKind, core.Float64NumberKind, opts))
}
