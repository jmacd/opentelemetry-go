package trace

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

type SpanReader interface {
	// Register is exported, e.g., so that metrics SDK can provide
	// a span-to-metrics producer as span reader, logs SDK can
	// ... etc.  This differs from the metric SDK because the
	// metric SDK reader is substantially more complex.
	Register(*resource.Resource)

	SpanOnStarter
	SpanOnAddLinker
	SpanOnEnder
	SpanWriteOnEnder
	Component
}

type SpanViewer interface {
}

func NewSpanReader(sa ComposableSampler, sp SpanProcessor) SpanReader {
	// return a span reader that sends spans to an exporter
	// includes a composite sampler
	return &spanReader{
		sampler:   sa,
		processor: sp,
	}
}

type spanReader struct {
	sampler   ComposableSampler
	processor SpanProcessor

	shutdownOnce sync.Once

	onStart   []SpanOnStarter
	onAddLink []SpanOnAddLinker
	onEndW    []SpanWriteOnEnder
	onEndR    []SpanOnEnder
}

var _ SpanReader = &spanReader{}

func (sr *spanReader) Register(*resource.Resource) {
	// @@@
}

func (sr *spanReader) OnStart(parent context.Context, s ReadWriteSpan) {
	sr.processor.OnStart(parent, s)
	// @@@
}

func (sr *spanReader) OnAddLink(trace.Link, ReadWriteSpan) {
	// @@@
}

func (sr *spanReader) OnEnd(s ReadOnlySpan) {
	sr.processor.OnEnd(s)
	// @@@
}

func (sr *spanReader) WriteOnEnd(s ReadWriteSpan) {
	// @@@
}

func (sr *spanReader) Shutdown(ctx context.Context) error {
	err := fmt.Errorf("already shutdown")
	sr.shutdownOnce.Do(func() {
		err = sr.processor.Shutdown(ctx)
	})
	return err
}

func (sr *spanReader) ForceFlush(ctx context.Context) error {
	return sr.processor.ForceFlush(ctx)
}
