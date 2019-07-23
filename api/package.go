package opentelemetry

import (
	"sync"

	"go.opentelemetry.io/api/internal"
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
		internal.GlobalTracer.Store(sdk.(trace.Tracer))
		internal.GlobalMeter.Store(sdk.(metric.Meter))
	})
}
