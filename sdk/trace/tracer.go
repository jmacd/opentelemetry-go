// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/otel/sdk/trace"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

type tracer struct {
	embedded.Tracer

	provider             *TracerProvider
	instrumentationScope instrumentation.Scope
}

var _ trace.Tracer = &tracer{}

// Start starts a Span and returns it along with a context containing it.
//
// The Span is created with the provided name and as a child of any existing
// span context found in the passed context. The created Span will be
// configured appropriately by any SpanOption passed.
func (tr *tracer) Start(ctx context.Context, name string, options ...trace.SpanStartOption) (context.Context, trace.Span) {
	config := trace.NewSpanStartConfig(options...)

	if ctx == nil {
		// Prevent trace.ContextWithSpan from panicking.
		ctx = context.Background()
	}

	// For local spans created by this SDK, track child span count.
	if p := trace.SpanFromContext(ctx); p != nil {
		if sdkSpan, ok := p.(*recordingSpan); ok {
			sdkSpan.addChild()
		}
	}

	s := tr.newSpan(ctx, name, &config)
	if rtt, ok := s.(runtimeTracer); ok {
		ctx = rtt.runtimeTrace(ctx)
	}

	return trace.ContextWithSpan(ctx, s), s
}

type runtimeTracer interface {
	// runtimeTrace starts a "runtime/trace".Task for the span and
	// returns a context containing the task.
	runtimeTrace(ctx context.Context) context.Context
}

// newSpan returns a new configured span.
func (tr *tracer) newSpan(ctx context.Context, name string, config *trace.SpanConfig) trace.Span {
	// If told explicitly to make this a new root use a zero value SpanContext
	// as a parent which contains an invalid trace ID and is not remote.
	var psc trace.SpanContext
	if config.NewRoot() {
		ctx = trace.ContextWithSpanContext(ctx, psc)
	} else {
		psc = trace.SpanContextFromContext(ctx)
	}

	// If there is a valid parent trace ID, use it to ensure the continuity of
	// the trace. Always generate a new span ID so other components can rely
	// on a unique span ID, even if the Span is non-recording.
	var tid trace.TraceID
	var sid trace.SpanID
	if !psc.TraceID().IsValid() {
		tid, sid = tr.provider.idGenerator.NewIDs(ctx)
	} else {
		tid = psc.TraceID()
		sid = tr.provider.idGenerator.NewSpanID(ctx, tid)
	}

	pipes := tr.provider.getPipelines()
	result, params := pipes.shouldSample(
		ctx, psc, tr.instrumentationScope, tid, sid, name, config,
	)

	scc := trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceState: result.Tracestate,
	}
	if isSampled(result) {
		scc.TraceFlags = psc.TraceFlags() | trace.FlagsSampled
	} else {
		scc.TraceFlags = psc.TraceFlags() &^ trace.FlagsSampled
	}
	sc := trace.NewSpanContext(scc)

	if !isRecording(result) {
		return tr.newNonRecordingSpan(sc)
	}
	s := tr.newRecordingSpan(psc, sc, name, result, config)
	for _, r := range pipes.readers {
		r.OnStart(ctx, s)
	}
	for _, r := range pipes.readers {
		r.OnSample(params, result.Decision)
	}
	return s
}

// newRecordingSpan returns a new configured recordingSpan.
func (tr *tracer) newRecordingSpan(psc, sc trace.SpanContext, name string, sr SamplingResult, config *trace.SpanConfig) *recordingSpan {
	startTime := config.Timestamp()
	if startTime.IsZero() {
		startTime = time.Now()
	}

	s := &recordingSpan{
		// Do not pre-allocate the attributes slice here! Doing so will
		// allocate memory that is likely never going to be used, or if used,
		// will be over-sized. The default Go compiler has been tested to
		// dynamically allocate needed space very well. Benchmarking has shown
		// it to be more performant than what we can predetermine here,
		// especially for the common use case of few to no added
		// attributes.

		parent:      psc,
		spanContext: sc,
		spanKind:    trace.ValidateSpanKind(config.SpanKind()),
		name:        name,
		startTime:   startTime,
		events:      newEvictedQueue(tr.provider.spanLimits.EventCountLimit),
		links:       newEvictedQueue(tr.provider.spanLimits.LinkCountLimit),
		tracer:      tr,
	}

	for _, l := range config.Links() {
		s.AddLink(l)
	}

	s.SetAttributes(sr.Attributes...)
	s.SetAttributes(config.Attributes()...)

	return s
}

// newNonRecordingSpan returns a new configured nonRecordingSpan.
func (tr *tracer) newNonRecordingSpan(sc trace.SpanContext) nonRecordingSpan {
	return nonRecordingSpan{tracer: tr, sc: sc}
}
