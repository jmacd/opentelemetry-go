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

package sdk

import (
	"context"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/api/metric"
	"go.opentelemetry.io/experimental/streaming/exporter/observer"
)

type float64Gauge struct {
	handle *metric.Float64GaugeHandle
	scope  observer.ScopeID
}

type float64Cumulative struct {
	handle *metric.Float64CumulativeHandle
	scope  observer.ScopeID
}

type float64Additive struct {
	handle *metric.Float64AdditiveHandle
	scope  observer.ScopeID
}

type float64Measure struct {
	handle *metric.Float64MeasureHandle
	scope  observer.ScopeID
}

func (s *sdk) GetFloat64Gauge(ctx context.Context, handle *metric.Float64GaugeHandle, labels ...core.KeyValue) metric.Float64Gauge {
	return &float64Gauge{
		handle: handle,
		scope:  observer.NewScope(observer.ScopeID{}, labels...),
	}
}

func (s *sdk) GetFloat64Cumulative(ctx context.Context, handle *metric.Float64CumulativeHandle, labels ...core.KeyValue) metric.Float64Cumulative {
	return &float64Cumulative{
		handle: handle,
		scope:  observer.NewScope(observer.ScopeID{}, labels...),
	}
}

func (s *sdk) GetFloat64Additive(ctx context.Context, handle *metric.Float64AdditiveHandle, labels ...core.KeyValue) metric.Float64Additive {
	return &float64Additive{
		handle: handle,
		scope:  observer.NewScope(observer.ScopeID{}, labels...),
	}
}

func (s *sdk) GetFloat64Measure(ctx context.Context, handle *metric.Float64MeasureHandle, labels ...core.KeyValue) metric.Float64Measure {
	return &float64Measure{
		handle: handle,
		scope:  observer.NewScope(observer.ScopeID{}, labels...),
	}
}

func record(ctx context.Context, handle metric.Handle, etype observer.EventType, value float64, scope observer.ScopeID, labels []core.KeyValue) {
	observer.Record(observer.Event{
		Type:    etype,
		Context: ctx,
		Measurement: observer.Measurement{
			Metric: handle,
			Value:  value,
			Scope:  scope,
		},
		Attributes: labels,
	})

}

func (g *float64Gauge) Set(ctx context.Context, value float64, labels ...core.KeyValue) {
	record(ctx, g.handle.Handle, observer.UPDATE_METRIC, value, g.scope, labels)
}

func (c *float64Cumulative) Inc(ctx context.Context, value float64, labels ...core.KeyValue) {
	record(ctx, c.handle.Handle, observer.UPDATE_METRIC, value, c.scope, labels)
}

func (a *float64Additive) Add(ctx context.Context, value float64, labels ...core.KeyValue) {
	record(ctx, a.handle.Handle, observer.UPDATE_METRIC, value, a.scope, labels)
}

func (m *float64Measure) Record(ctx context.Context, value float64, labels ...core.KeyValue) {
	record(ctx, m.handle.Handle, observer.UPDATE_METRIC, value, m.scope, labels)
}
