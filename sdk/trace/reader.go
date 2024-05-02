package trace

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/sdk/resource"
)

// SpanReader consists of a composable sampler, a span processor, and
// an OnSample method which allows per-pipeline customization of the
// Span data that is emitted into the processor.
type SpanReader interface {
	// ComposableSampler includes Description(), Register(), and
	// ShouldSample().
	ComposableSampler

	// SpanProcessor includes OnStart(), OnEnd(), ForceFlush(),
	// Shutdown().
	SpanProcessor

	// OnSample is called following a decision to sample, receives
	// the original parameters, the composite decision and returns
	// a pipeline-specific SpanViewer.
	OnSample(SamplingParameters2, SamplingDecision) SpanViewer
}

// NewSimpleSpanReader returns a simple span reader consisting of just
// a sampler and processor, with no additional OnSample functionality.
func NewSimpleSpanReader(sa ComposableSampler, sp SpanProcessor) SpanReader {
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
}

var _ SpanReader = &spanReader{}

func (sr *spanReader) Register(res *resource.Resource) {
	sr.sampler.Register(res)
}

func (sr *spanReader) Description() string {
	return sr.sampler.Description()
}

func (sr *spanReader) OnStart(parent context.Context, s ReadWriteSpan) {
	sr.processor.OnStart(parent, s)
}

func (sr *spanReader) OnEnd(s ReadOnlySpan) {
	sr.processor.OnEnd(s)
}

func (sr *spanReader) ShouldSample(params SamplingParameters2) (SamplingDecision, Threshold) {
	return sr.sampler.ShouldSample(params)
}

func (sr *spanReader) OnSample(SamplingParameters2, SamplingDecision) SpanViewer {
	return nil
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
