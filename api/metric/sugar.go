package metric

import (
	"context"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/internal"
)

type provider interface {
	Meter() Meter
}

func getMeter(ctx context.Context) Meter {
	if p, ok := internal.ScopeImpl(ctx).(provider); ok {
		return p.Meter()
	}

	if g, ok := internal.GlobalScope.Load().(provider); ok {
		return g.Meter()
	}

	return NoopMeter{}
}

func NewInt64Counter(ctx context.Context, name string, cos ...CounterOptionApplier) Int64Counter {
	meter := getMeter(ctx)
	return meter.NewInt64Counter(ctx, name, cos...)
}

func NewFloat64Counter(ctx context.Context, name string, cos ...CounterOptionApplier) Float64Counter {
	meter := getMeter(ctx)
	return meter.NewFloat64Counter(ctx, name, cos...)
}

func NewInt64Gauge(ctx context.Context, name string, gos ...GaugeOptionApplier) Int64Gauge {
	meter := getMeter(ctx)
	return meter.NewInt64Gauge(ctx, name, gos...)
}

func NewFloat64Gauge(ctx context.Context, name string, gos ...GaugeOptionApplier) Float64Gauge {
	meter := getMeter(ctx)
	return meter.NewFloat64Gauge(ctx, name, gos...)
}

func NewInt64Measure(ctx context.Context, name string, mos ...MeasureOptionApplier) Int64Measure {
	meter := getMeter(ctx)
	return meter.NewInt64Measure(ctx, name, mos...)
}

func NewFloat64Measure(ctx context.Context, name string, mos ...MeasureOptionApplier) Float64Measure {
	meter := getMeter(ctx)
	return meter.NewFloat64Measure(ctx, name, mos...)
}

func RecordBatch(ctx context.Context, labels []core.KeyValue, ms ...Measurement) {
	meter := getMeter(ctx)
	meter.RecordBatch(ctx, labels, ms...)
}
