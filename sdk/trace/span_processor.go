// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/otel/sdk/trace"

import (
	"context"
	"sync"
)

type SpanMutator interface {

	// OnStart is called when a span is started. It is called synchronously
	// and should not block.
	OnStart(parent context.Context, s ReadWriteSpan)
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.
}

type ReadOnEndSpanProcessor interface {
	SpanMutator

	// OnEnd is called when span is finished. It is called synchronously and
	// hence not block.
	OnEnd(s ReadOnlySpan)
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.
}

type WriteOnEndSpanProcessor interface {
	SpanMutator

	// OnEnd is called when span is finished. It is called synchronously and
	// hence not block.
	OnEnd(s ReadWriteSpan)
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.
}

type Component interface {
	// Shutdown is called when the SDK shuts down. Any cleanup or release of
	// resources held by the processor should be done in this call.
	//
	// Calls to OnStart, OnEnd, or ForceFlush after this has been called
	// should be ignored.
	//
	// All timeouts and cancellations contained in ctx must be honored, this
	// should not block indefinitely.
	Shutdown(ctx context.Context) error
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.

	// ForceFlush exports all ended spans to the configured Exporter that have not yet
	// been exported.  It should only be called when absolutely necessary, such as when
	// using a FaaS provider that may suspend the process after an invocation, but before
	// the Processor can export the completed spans.
	ForceFlush(ctx context.Context) error
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.
}

// SpanProcessor is a processing pipeline for spans in the trace signal.
// SpanProcessors registered with a TracerProvider and are called at the start
// and end of a Span's lifecycle, and are called in the order they are
// registered.
type SpanProcessor interface {
	ReadOnEndSpanProcessor

	Component
}

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

func (p pipelines) getSampler() Sampler {
	// @@@ NOT
	return nil
}

func processorToReader(sp SpanProcessor) SpanReader {
	return &processorReader{
		processor: sp,
	}
}

type processorReader struct {
	processor SpanProcessor
	once      sync.Once // @@@
}

var _ SpanReader = &processorReader{}

func (pr *processorReader) register() {
}

func (pr *processorReader) OnStart(parent context.Context, s ReadWriteSpan) {
	pr.processor.OnStart(parent, s)
}

func (pr *processorReader) OnEnd(s ReadWriteSpan) {
	pr.processor.OnEnd(s.(ReadOnlySpan))
}

func (pr *processorReader) Shutdown(ctx context.Context) error {
	return pr.processor.Shutdown(ctx)
}

func (pr *processorReader) ForceFlush(ctx context.Context) error {
	return pr.processor.ForceFlush(ctx)
}
