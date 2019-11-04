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

package export

import (
	"context"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/unit"
)

// MetricAggregationSelector supports selecting the kind of aggregator
// to use at runtime for a specific metric instrument.
type MetricAggregationSelector interface {
	// AggregatorFor should return the kind of aggregator suited
	// to the requested export.  Returning `nil` indicates to
	// ignore this metric instrument.  Although it is not
	// required, this should return a consistent type to avoid
	// confusion in later stages of the metrics export process.
	//
	// Note: This is context-free because the handle should not be
	// bound to the incoming context.  This call should not block.
	AggregatorFor(MetricRecord) MetricAggregator
}

// MetricAggregator implements a specific aggregation behavior, e.g.,
// a counter, a gauge, a histogram.
type MetricAggregator interface {
	// Update receives a new measured value and incorporates it
	// into the aggregation.
	Update(context.Context, core.Number, MetricRecord)

	// Collect is called during the SDK Collect() to
	// finish one period of aggregation.  Collect() is
	// called in a single-threaded context.  Update()
	// calls may arrive concurrently.
	Collect(context.Context, MetricRecord, MetricBatcher)

	// Merge combines state from two aggregators into one.
	Merge(MetricAggregator, *Descriptor)
}

// MetricRecord is the unit of export, pairing a metric
// instrument and set of labels.
type MetricRecord interface {
	// Descriptor() describes the metric instrument.
	Descriptor() *Descriptor

	// Labels() describe the labsels corresponding the
	// aggregation being performed.
	Labels() []core.KeyValue

	// EncodedLabels are a unique string-encoded form of Labels()
	// suitable for use as a map key.
	EncodedLabels() string
}

// MetricExporter handles presentation of the checkpoint of aggregate
// metrics.  This is the final stage of a metrics export pipeline,
// where metric data are formatted for a specific system.
type MetricExporter interface {
	Export(context.Context, MetricProducer)
}

// MetricLabelEncoder enables an optimization for export pipelines that
// use text to encode their label sets.  This interface allows configuring
// the encoder used in the SDK and/or the MetricBatcher so that by the
// time the exporter is called, the same encoding may be used.
type MetricLabelEncoder interface {
	EncodeLabels([]core.KeyValue) string
}

// ProducedRecord
type ProducedRecord struct {
	Descriptor    *Descriptor
	Labels        []core.KeyValue
	Encoder       MetricLabelEncoder
	EncodedLabels string
}

// MetricProducer allows a MetricExporter to access a checkpoint of
// aggregated metrics one at a time.
type MetricProducer interface {
	Foreach(func(MetricAggregator, ProducedRecord))
}

// MetricKind describes the kind of instrument.
type MetricKind int8

const (
	CounterMetricKind MetricKind = iota
	GaugeMetricKind
	MeasureMetricKind
)

// Descriptor describes a metric instrument to the exporter.
type Descriptor struct {
	name        string
	metricKind  MetricKind
	keys        []core.Key
	description string
	unit        unit.Unit
	numberKind  core.NumberKind
	alternate   bool
}

// NewDescriptor builds a new descriptor, for use by `Meter`
// implementations.
func NewDescriptor(
	name string,
	metricKind MetricKind,
	keys []core.Key,
	description string,
	unit unit.Unit,
	numberKind core.NumberKind,
	alternate bool,
) *Descriptor {
	return &Descriptor{
		name:        name,
		metricKind:  metricKind,
		keys:        keys,
		description: description,
		unit:        unit,
		numberKind:  numberKind,
		alternate:   alternate,
	}
}

func (d *Descriptor) Name() string {
	return d.name
}

func (d *Descriptor) MetricKind() MetricKind {
	return d.metricKind
}

func (d *Descriptor) Keys() []core.Key {
	return d.keys
}

func (d *Descriptor) Description() string {
	return d.description
}

func (d *Descriptor) Unit() unit.Unit {
	return d.unit
}

func (d *Descriptor) NumberKind() core.NumberKind {
	return d.numberKind
}

func (d *Descriptor) Alternate() bool {
	return d.alternate
}
