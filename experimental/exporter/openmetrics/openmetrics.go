package openmetrics

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/sdk/export"
	"go.opentelemetry.io/sdk/metric/aggregator/counter"
	"go.opentelemetry.io/sdk/metric/aggregator/gauge"
	"go.opentelemetry.io/sdk/metric/aggregator/maxsumcount"
)

type (
	Exporter struct {
		dki dkiMap
		agg aggMap
		tmp [32]byte
	}

	aggEntry struct {
		aggregator export.MetricAggregator
		descriptor *export.Descriptor
	}

	dkiMap map[*export.Descriptor]map[core.Key]int
	aggMap map[string]aggEntry
)

var _ export.MetricBatcher = &Exporter{}

func NewExporter() *Exporter {
	return &Exporter{
		dki: dkiMap{},
		agg: aggMap{},
	}
}

func (e *Exporter) AggregatorFor(record export.MetricRecord) export.MetricAggregator {
	switch record.Descriptor().MetricKind() {
	case export.CounterMetricKind:
		return counter.New()
	case export.GaugeMetricKind:
		return gauge.New()
	case export.MeasureMetricKind:
		return maxsumcount.New()
	default:
		panic("Unknown metric kind")
	}
}

func (e *Exporter) Export(_ context.Context, record export.MetricRecord, agg export.MetricAggregator) {
	desc := record.Descriptor()
	keys := desc.Keys()

	// Cache the mapping from Descriptor->Key->Index
	ki, ok := e.dki[desc]
	if !ok {
		ki = map[core.Key]int{}
		e.dki[desc] = ki

		for i, k := range keys {
			ki[k] = i
		}
	}

	// TODO can add a cache of (LabelSet,Descriptor)->(Encoded).
	// Can use reflection to build right-sized arrays, or impose a
	// max-keys and use fixed-size arrays to speed this up.

	// Compute the value list.  TODO: Unspecified values become
	// empty strings, OK?
	canon := make([]core.Value, len(keys))

	for _, kv := range record.Labels() {
		pos, ok := ki[kv.Key]
		if !ok {
			continue
		}
		canon[pos] = kv.Value
	}

	// Compute an encoded lookup key
	var sb strings.Builder
	for i := 0; i < len(keys); i++ {
		sb.WriteString(string(keys[i]))
		sb.WriteRune('=')

		sb.WriteRune('"')
		canon[i].Encode(&sb, e.tmp[:])
		sb.WriteRune('"')

		if i < len(keys)-1 {
			sb.WriteRune(',')
		}
	}

	encoded := sb.String()

	// Perform the group-by.
	rag, ok := e.agg[encoded]
	if !ok {
		e.agg[encoded] = aggEntry{
			aggregator: agg,
			descriptor: record.Descriptor(),
		}
	} else {
		rag.aggregator.Merge(agg, record.Descriptor())
	}
}

func writePrefix(w core.Encoder, name string, labels string) {
	// E.g., `name{label="0"} `

	w.WriteString(name)
	if labels != "" {
		w.WriteRune('{')
		w.WriteString(labels)
		w.WriteRune('}')
	}
	w.WriteRune(' ')
}

func addQuantile(labels string, quantile float64) string {
	ql := fmt.Sprintf("quantile=\"%g\"", quantile)
	if labels == "" {
		return ql
	}
	return labels + "," + ql

}

func (e *Exporter) Write(w core.Encoder) (n int, err error) {

	for labels, entry := range e.agg {

		// TODO move these blocks of code into the aggregators?
		switch entry.descriptor.MetricKind() {
		case export.CounterMetricKind:
			cnt := entry.aggregator.(*counter.Aggregator)
			sum := cnt.AsNumber()

			writePrefix(w, entry.descriptor.Name(), labels)
			sum.Encode(entry.descriptor.NumberKind(), w, e.tmp[:])

		case export.GaugeMetricKind:
			gau := entry.aggregator.(*gauge.Aggregator)
			current := gau.AsNumber()

			writePrefix(w, entry.descriptor.Name(), labels)
			current.Encode(entry.descriptor.NumberKind(), w, e.tmp[:])

		case export.MeasureMetricKind:
			mea := entry.aggregator.(*maxsumcount.Aggregator)
			max := mea.Max()
			sum := mea.Sum()
			count := mea.Count()

			writePrefix(w, entry.descriptor.Name(), addQuantile(labels, 1))
			max.Encode(entry.descriptor.NumberKind(), w, e.tmp[:])
			w.WriteRune('\n')

			writePrefix(w, entry.descriptor.Name()+"_sum", labels)
			sum.Encode(entry.descriptor.NumberKind(), w, e.tmp[:])
			w.WriteRune('\n')
			writePrefix(w, entry.descriptor.Name()+"_count", labels)
			count.Encode(core.Uint64NumberKind, w, e.tmp[:])
		}

		w.WriteRune('\n')
	}

	e.Reset()

	return 0, nil
}

func (e *Exporter) Reset() {
	e.agg = aggMap{}
}
