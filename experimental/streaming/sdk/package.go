package sdk

import (
	"go.opentelemetry.io/api/metric"
	"go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/experimental/streaming/exporter/observer"
)

type sdk struct {
	resources observer.EventID
}

type SDK interface {
	trace.Tracer
	metric.Meter
}

var _ SDK = (*sdk)(nil)

func init() {
	instance := New()
	trace.SetGlobalTracer(instance)
	metric.SetGlobalMeter(instance)
}
