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

package metric

import (
	"context"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/api/metric/aggregation"
	"go.opentelemetry.io/api/registry"
	"go.opentelemetry.io/api/unit"
)

type Type int

//go:generate stringer -type=Type
const (
	Invalid    Type = iota
	Gauge           // Supports Set()
	Cumulative      // Supports Inc(): only positive values
	Additive        // Supports Add(): positive or negative
	Measure         // Supports Record()
)

type Meter interface {
	GetFloat64Gauge(context.Context, *Float64GaugeHandle, ...core.KeyValue) Float64Gauge
	GetFloat64Cumulative(context.Context, *Float64CumulativeHandle, ...core.KeyValue) Float64Cumulative
	GetFloat64Additive(context.Context, *Float64AdditiveHandle, ...core.KeyValue) Float64Additive
	GetFloat64Measure(context.Context, *Float64MeasureHandle, ...core.KeyValue) Float64Measure
}

type Float64Gauge interface {
	Set(ctx context.Context, value float64, labels ...core.KeyValue)
}

type Float64Cumulative interface {
	Inc(ctx context.Context, value float64, labels ...core.KeyValue)
}

type Float64Additive interface {
	Add(ctx context.Context, value float64, labels ...core.KeyValue)
}

type Float64Measure interface {
	Record(ctx context.Context, value float64, labels ...core.KeyValue)
}

type Handle struct {
	Variable registry.Variable

	Type        Type
	Keys        []core.Key
	Aggregation aggregation.Operator
}

type Option func(*Handle, *[]registry.Option)

// WithDescription applies provided description.
func WithDescription(desc string) Option {
	return func(_ *Handle, to *[]registry.Option) {
		*to = append(*to, registry.WithDescription(desc))
	}
}

// WithUnit applies provided unit.
func WithUnit(unit unit.Unit) Option {
	return func(_ *Handle, to *[]registry.Option) {
		*to = append(*to, registry.WithUnit(unit))
	}
}

// WithKeys applies recommended dimension keys.  Multiple `WithKeys`
// options accumulate.  The keys specified in this way are taken as
// the recommended aggregation keys for Gauge, Cumulative, and
// Additive metrics.  For Measure metrics, the keys recommended here
// are taken as the default for aggregations.
func WithKeys(keys ...core.Key) Option {
	return func(m *Handle, _ *[]registry.Option) {
		m.Keys = append(m.Keys, keys...)
	}
}

// WithAggregation applies a user-recommended aggregation to this
// metric, useful particularly for Measure type metrics.
func WithAggregation(aggr aggregation.Operator) Option {
	return func(m *Handle, _ *[]registry.Option) {
		m.Aggregation = aggr
	}
}

func (h Handle) Defined() bool {
	return h.Variable.Defined()
}
