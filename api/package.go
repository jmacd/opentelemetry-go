package opentelemetry

import (
	"sync"

	"go.opentelemetry.io/api/metric"
	"go.opentelemetry.io/api/trace"
)

var once sync.Once

type SDK interface {
	trace.Tracer
	metric.Meter
}

func Init(sdk SDK) {
	once.Do(func() {
		// internal.GlobalTracer.Store(sdk.Tracer)
		// internal.GlobalMeter.Store(sdk.Meter)
	})
}
