package sdk

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-go/api/core"
	"github.com/open-telemetry/opentelemetry-go/api/event"
	"github.com/open-telemetry/opentelemetry-go/api/loader"
	"github.com/open-telemetry/opentelemetry-go/api/metric"
	"github.com/open-telemetry/opentelemetry-go/api/stats"
	"github.com/open-telemetry/opentelemetry-go/api/tag"
	"github.com/open-telemetry/opentelemetry-go/api/trace"
	"google.golang.org/grpc/codes"
)

type SDK struct {
}

type span struct {
	sdk         *SDK
	spanContext core.SpanContext
}

type gauge struct {
}

var _ trace.Tracer = (*SDK)(nil)
var _ metric.Meter = (*SDK)(nil)
var _ stats.Recorder = (*SDK)(nil)
var _ loader.Flusher = (*SDK)(nil)

var _ trace.Span = (*span)(nil)

var _ metric.Float64Gauge = (*gauge)(nil)

func New() *SDK {
	return &SDK{}
}

func (sdk *SDK) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	fmt.Println("Start")
	return ctx, &span{sdk: sdk}
}

func (sdk *SDK) WithSpan(
	ctx context.Context,
	operation string,
	body func(ctx context.Context) error) error {
	ctx, _ = sdk.Start(ctx, operation)
	return body(ctx)
}

func (sdk *SDK) WithService(name string) trace.Tracer {
	fmt.Println("WithService")
	return sdk
}
func (sdk *SDK) WithComponent(name string) trace.Tracer {
	fmt.Println("WithComponent")
	return sdk
}

func (sdk *SDK) WithResources(res ...core.KeyValue) trace.Tracer {
	fmt.Println("WithResources")
	return sdk
}

func (sdk *SDK) Inject(context.Context, trace.Span, trace.Injector) {
	fmt.Println("Inject")
}

func (sdk *SDK) GetFloat64Gauge(ctx context.Context, handle *metric.Float64GaugeHandle, labels ...core.KeyValue) metric.Float64Gauge {
	fmt.Println("GetFloat64Gauge")
	return &gauge{}
}

func (sdk *SDK) GetMeasure(ctx context.Context, measure *stats.MeasureHandle, labels ...core.KeyValue) stats.Measure {
	fmt.Println("GetMeasure")
	return &stats.MeasureHandle{}
}

func (sdk *SDK) Record(ctx context.Context, m ...stats.Measurement) {
	fmt.Println("Record")
}

func (sdk *SDK) RecordSingle(ctx context.Context, m stats.Measurement) {
	fmt.Println("RecordSingle")
}

func (sdk *SDK) Flush() error {
	fmt.Println("Flush")
	return nil
}

func (sp *span) Tracer() trace.Tracer {
	return sp.sdk
}

func (sp *span) Finish() {
	fmt.Println("Finish")
}

func (sp *span) AddEvent(ctx context.Context, event event.Event) {
	fmt.Println("Event")
}

func (sp *span) Event(ctx context.Context, msg string, attrs ...core.KeyValue) {
	fmt.Println("Event")
}

func (sp *span) IsRecordingEvents() bool {
	return false
}

func (sp *span) SpanContext() core.SpanContext {
	return sp.spanContext
}

func (sp *span) SetStatus(codes.Code) {
	fmt.Println("SetStatus")
}

func (sp *span) SetAttribute(core.KeyValue) {
	fmt.Println("SetAttribute")
}
func (sp *span) SetAttributes(...core.KeyValue) {
	fmt.Println("SetAttributes")
}

func (sp *span) ModifyAttribute(tag.Mutator) {
	fmt.Println("SetAttribute")
}
func (sp *span) ModifyAttributes(...tag.Mutator) {
	fmt.Println("SetAttributes")
}

func (g *gauge) Set(ctx context.Context, value float64, labels ...core.KeyValue) {
	fmt.Println("Gauge.Set")
}
