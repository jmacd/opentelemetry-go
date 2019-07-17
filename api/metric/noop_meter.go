package metric

import (
	"context"

	"go.opentelemetry.io/api/core"
)

type noopMeter struct{}

type noopMetric struct{}

var _ Meter = noopMeter{}

var _ Float64Gauge = noopMetric{}
var _ Float64Cumulative = noopMetric{}
var _ Float64Additive = noopMetric{}
var _ Float64Measure = noopMetric{}

func (noopMeter) GetFloat64Gauge(ctx context.Context, gauge *Float64GaugeHandle, labels ...core.KeyValue) Float64Gauge {
	return noopMetric{}
}

func (noopMeter) GetFloat64Cumulative(ctx context.Context, gauge *Float64CumulativeHandle, labels ...core.KeyValue) Float64Cumulative {
	return noopMetric{}
}

func (noopMeter) GetFloat64Additive(ctx context.Context, gauge *Float64AdditiveHandle, labels ...core.KeyValue) Float64Additive {
	return noopMetric{}
}

func (noopMeter) GetFloat64Measure(ctx context.Context, gauge *Float64MeasureHandle, labels ...core.KeyValue) Float64Measure {
	return noopMetric{}
}

func (noopMetric) Set(ctx context.Context, value float64, labels ...core.KeyValue) {
}

func (noopMetric) Inc(ctx context.Context, value float64, labels ...core.KeyValue) {
}

func (noopMetric) Add(ctx context.Context, value float64, labels ...core.KeyValue) {
}

func (noopMetric) Record(ctx context.Context, value float64, labels ...core.KeyValue) {
}
