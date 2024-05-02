package trace

import (
	"context"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/trace"
)

// pipelines are immutable
type pipelines struct {
	provider *TracerProvider
	readers  []SpanReader
}

func (provider *TracerProvider) newPipelines(readers []SpanReader) *pipelines {
	return &pipelines{
		provider: provider,
		readers:  readers,
	}
}

func (p pipelines) add(r SpanReader) *pipelines {
	rs := make([]SpanReader, len(p.readers)+1)
	copy(rs[:len(p.readers)], p.readers)
	rs[len(p.readers)] = r
	return p.provider.newPipelines(rs)
}

func (p pipelines) remove(remFunc func(SpanReader) bool) *pipelines {
	rs := make([]SpanReader, 0, len(p.readers))
	for _, r := range p.readers {
		if !remFunc(r) {
			rs = append(rs, r)
		}
	}
	return p.provider.newPipelines(rs)
}

// shouldSample receives all the relevant inputs during the start of a
// new span
func (p pipelines) shouldSample(
	ctx context.Context,
	parent trace.SpanContext,
	scope instrumentation.Scope,
	tid trace.TraceID,
	sid trace.SpanID,
	name string,
	config *trace.SpanConfig,
) (SamplingResult, SamplingParameters2) {

	params1 := SamplingParameters{
		ParentContext: ctx,
		TraceID:       tid,
		Name:          name,
		Kind:          config.SpanKind(),
		Attributes:    config.Attributes(),
		Links:         config.Links(),
	}
	params2 := SamplingParameters2{
		parameters: params1,
		scope:      scope,
		spanID:     sid,
		parent:     parent,
	}

	// @@@ Would like to make this a params2 call, w/ a wrapper.
	// This appears possible, but not straightforward because it
	// can't simply use the ComposableSampler interface w/o
	// somehow also modifying the span attributes and t-value.
	primeResult := p.provider.sampler.ShouldSample(params1)
	if primeResult.Decision == Drop {
		return primeResult, params2
	}

	// @@@ Implement.
	return primeResult, params2
}
