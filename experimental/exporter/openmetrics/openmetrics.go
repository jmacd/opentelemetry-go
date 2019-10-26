package openmetrics

import (
	"context"
	"io"

	"go.opentelemetry.io/sdk/export"
	"go.opentelemetry.io/sdk/metric/aggregator/array"
	"go.opentelemetry.io/sdk/metric/aggregator/counter"
	"go.opentelemetry.io/sdk/metric/aggregator/gauge"
)

type (
	OpenMetricsExporter struct {
	}
)

var _ export.MetricBatcher = &OpenMetricsExporter{}

func NewOpenMetricsExporter() *OpenMetricsExporter {
	return &OpenMetricsExporter{}
}

func (e *OpenMetricsExporter) AggregatorFor(record export.MetricRecord) export.MetricAggregator {
	switch record.Descriptor().MetricKind() {
	case export.CounterMetricKind:
		return counter.New()
	case export.GaugeMetricKind:
		return gauge.New()
	case export.MeasureMetricKind:
		return array.New()
	default:
		panic("Unknown metric kind")
	}
}

func (e *OpenMetricsExporter) Export(_ context.Context, record export.MetricRecord, agg export.MetricAggregator) {

}

func (e *OpenMetricsExporter) Write(w io.Writer) (n int, err error) {
	return 0, nil
}
