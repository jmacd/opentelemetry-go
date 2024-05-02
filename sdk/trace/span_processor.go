// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/otel/sdk/trace"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanProcessor is a processing pipeline for spans in the trace signal.
// SpanProcessors registered with a TracerProvider and are called at the start
// and end of a Span's lifecycle, and are called in the order they are
// registered.
type SpanProcessor interface {
	// OnStart is called when a span is started. It is called synchronously
	// and should not block.
	OnStart(parent context.Context, s ReadWriteSpan)
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.

	// OnEnd is called when span is finished. It is called synchronously and
	// hence not block.
	OnEnd(ReadOnlySpan)
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.

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

// SpanViewer is returned from the SpanReader.OnSample callback.  This
// gives registered SpanReaders an opportunity to observe each span
// event, build state about those observations specific to the
// pipeline, and change the per-pipeline sampling decision on-the-fly.
type SpanViewer interface {
	// Each of the On*() methods is called by the SDK on the
	// per-reader SpanViewer interface.  Each allows the
	// per-reader sampling decision to be modified.

	OnSetName(name string) SamplingDecision
	OnAddLink(link trace.Link) SamplingDecision
	OnSetAttributes(attrs []attribute.KeyValue) SamplingDecision
	OnAddEvent(name string, evt trace.EventConfig)
	OnSetStatus(code codes.Code, msg string)

	// ModifyOnEnd is called with the finalized read-only span
	// calculated by the SDK.  This gives the viewer an
	// opportunity to modify any aspects of the original span.
	// This is called prior to the SpanProcessor.OnEnd() method.
	ModifyOnEnd(final ReadOnlySpan) ReadOnlySpan
}
