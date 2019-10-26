package openmetrics

import (
	"context"
	"io"
	"strings"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/sdk/export"
	"go.opentelemetry.io/sdk/metric/aggregator/array"
	"go.opentelemetry.io/sdk/metric/aggregator/counter"
	"go.opentelemetry.io/sdk/metric/aggregator/gauge"
)

type (
	Exporter struct {
		dki dkiMap
	}

	dkiMap map[*export.Descriptor]map[core.Key]int
)

var _ export.MetricBatcher = &Exporter{}

func NewExporter() *Exporter {
	return &Exporter{
		dki: dkiMap{},
	}
}

func (e *Exporter) AggregatorFor(record export.MetricRecord) export.MetricAggregator {
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

func (e *Exporter) Export(_ context.Context, record export.MetricRecord, agg export.MetricAggregator) {
	desc := record.Descriptor()
	keys := desc.Keys()

	ki, ok := e.dki[desc]
	if !ok {
		ki = map[core.Key]int{}
		e.dki[desc] = ki

		for i, k := range keys {
			ki[k] = i
		}
	}

	canon := make([]core.Value, len(keys))

	for _, kv := range record.Labels() {
		pos, ok := ki[kv.Key]
		if !ok {
			continue
		}
		canon[pos] = kv.Value
	}

	var sb strings.Builder
	sb.WriteRune('{')

	for i := 0; i < len(keys); i++ {
		sb.WriteString(keys[i])
		sb.WriteRune('=')

		sb.WriteRune('"')
		canon[i].Encode(&sb)
		sb.WriteRune('"')

		if i < len(keys)-1 {
			sb.WriteRune(',')
		}
	}

	sb.WriteRune('}')

	// go_gc_duration_seconds{quantile="0"} 4.9351e-05
}

func (e *Exporter) Write(w io.Writer) (n int, err error) {
	return 0, nil
}
