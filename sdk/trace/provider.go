// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/otel/sdk/trace"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/internal/global"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	defaultTracerName = "go.opentelemetry.io/otel/sdk/tracer"
)

// tracerProviderConfig.
type tracerProviderConfig struct {
	// readers are {V2 sampler, write-on-end processor, span processor/exporter}
	readers []SpanReader

	// sampler is the TraceProvider-wide Sampler; attributes and
	// tracestate modifications apply to all span pipelines.
	sampler Sampler

	// idGenerator is used to generate all Span and Trace IDs when needed.
	idGenerator IDGenerator

	// spanLimits defines the attribute, event, and link limits for spans.
	spanLimits SpanLimits

	// resource contains attributes representing an entity that produces telemetry.
	resource *resource.Resource
}

// MarshalLog is the marshaling function used by the logging system to represent this Provider.
func (cfg tracerProviderConfig) MarshalLog() interface{} {
	return struct {
		Readers         []SpanReader
		SamplerType     string
		IDGeneratorType string
		SpanLimits      SpanLimits
		Resource        *resource.Resource
	}{
		Readers:         cfg.readers,
		SamplerType:     fmt.Sprintf("%T", cfg.sampler),
		IDGeneratorType: fmt.Sprintf("%T", cfg.idGenerator),
		SpanLimits:      cfg.spanLimits,
		Resource:        cfg.resource,
	}
}

// TracerProvider is an OpenTelemetry TracerProvider. It provides Tracers to
// instrumentation so it can trace operational flow through a system.
type TracerProvider struct {
	embedded.TracerProvider

	mu          sync.Mutex
	namedTracer map[instrumentation.Scope]*tracer
	pipes       atomic.Pointer[pipelines]

	isShutdown atomic.Bool

	// These fields are not protected by the lock mu. They are assumed to be
	// immutable after creation of the TracerProvider.
	sampler     Sampler
	idGenerator IDGenerator
	spanLimits  SpanLimits
	resource    *resource.Resource
}

var _ trace.TracerProvider = &TracerProvider{}

// NewTracerProvider returns a new and configured TracerProvider.
//
// By default the returned TracerProvider is configured with:
//   - a ParentBased(AlwaysSample) Sampler
//   - a random number IDGenerator
//   - the resource.Default() Resource
//   - the default SpanLimits.
//
// The passed opts are used to override these default values and configure the
// returned TracerProvider appropriately.
func NewTracerProvider(opts ...TracerProviderOption) *TracerProvider {
	o := tracerProviderConfig{
		spanLimits: NewSpanLimits(),
	}
	o = applyTracerProviderEnvConfigs(o)

	for _, opt := range opts {
		o = opt.apply(o)
	}

	o = ensureValidTracerProviderConfig(o)

	tp := &TracerProvider{
		namedTracer: make(map[instrumentation.Scope]*tracer),
		sampler:     o.sampler,
		idGenerator: o.idGenerator,
		spanLimits:  o.spanLimits,
		resource:    o.resource,
	}
	global.Info("TracerProvider created", "config", o)

	tp.pipes.Store(tp.newPipelines(o.readers))

	return tp
}

// Tracer returns a Tracer with the given name and options. If a Tracer for
// the given name and options does not exist it is created, otherwise the
// existing Tracer is returned.
//
// If name is empty, DefaultTracerName is used instead.
//
// This method is safe to be called concurrently.
func (p *TracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	// This check happens before the mutex is acquired to avoid deadlocking if Tracer() is called from within Shutdown().
	if p.isShutdown.Load() {
		return noop.NewTracerProvider().Tracer(name, opts...)
	}
	c := trace.NewTracerConfig(opts...)
	if name == "" {
		name = defaultTracerName
	}
	is := instrumentation.Scope{
		Name:      name,
		Version:   c.InstrumentationVersion(),
		SchemaURL: c.SchemaURL(),
	}

	t, ok := func() (trace.Tracer, bool) {
		p.mu.Lock()
		defer p.mu.Unlock()
		// Must check the flag after acquiring the mutex to avoid returning a valid tracer if Shutdown() ran
		// after the first check above but before we acquired the mutex.
		if p.isShutdown.Load() {
			return noop.NewTracerProvider().Tracer(name, opts...), true
		}
		t, ok := p.namedTracer[is]
		if !ok {
			t = &tracer{
				provider:             p,
				instrumentationScope: is,
			}
			p.namedTracer[is] = t
		}
		return t, ok
	}()
	if !ok {
		// This code is outside the mutex to not hold the lock while calling third party logging code:
		// - That code may do slow things like I/O, which would prolong the duration the lock is held,
		//   slowing down all tracing consumers.
		// - Logging code may be instrumented with tracing and deadlock because it could try
		//   acquiring the same non-reentrant mutex.
		global.Info("Tracer created", "name", name, "version", is.Version, "schemaURL", is.SchemaURL)
	}
	return t
}

// RegisterSpanProcessor adds the given SpanProcessor to the list of SpanProcessors.
func (p *TracerProvider) RegisterSpanProcessor(sp SpanProcessor) {
	p.RegisterSpanReader(NewSimpleSpanReader(nil, sp))
}

func (p *TracerProvider) RegisterSpanReader(sr SpanReader) {

	// This check prevents calls during a shutdown.
	if p.isShutdown.Load() {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// This check prevents calls after a shutdown.
	if p.isShutdown.Load() {
		return
	}

	p.pipes.Store(p.getPipelines().add(sr))
}

// UnregisterSpanProcessor removes the given SpanProcessor from the list of SpanProcessors.
func (p *TracerProvider) UnregisterSpanProcessor(sp SpanProcessor) {
	p.unregisterPipeline(func(r SpanReader) bool {
		if pr, ok := r.(*spanReader); ok {
			return pr.processor == sp
		}
		return false
	})
}

func (p *TracerProvider) UnregisterSpanReader(sr SpanReader) {
	p.unregisterPipeline(func(r SpanReader) bool {
		return sr == r
	})
}

func (p *TracerProvider) unregisterPipeline(f func(r SpanReader) bool) {
	// This check prevents calls during a shutdown.
	if p.isShutdown.Load() {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// This check prevents calls after a shutdown.
	if p.isShutdown.Load() {
		return
	}
	p.pipes.Store(p.getPipelines().remove(f))
}

// ForceFlush immediately exports all spans that have not yet been exported for
// all the registered span processors.
func (p *TracerProvider) ForceFlush(ctx context.Context) error {
	ps := p.getPipelines()
	if len(ps.readers) == 0 {
		return nil
	}

	for _, r := range ps.readers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := r.ForceFlush(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Shutdown shuts down TracerProvider. All registered span processors are shut down
// in the order they were registered and any held computational resources are released.
// After Shutdown is called, all methods are no-ops.
func (p *TracerProvider) Shutdown(ctx context.Context) error {
	// This check prevents deadlocks in case of recursive shutdown.
	if p.isShutdown.Load() {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// This check prevents calls after a shutdown has already been done concurrently.
	if !p.isShutdown.CompareAndSwap(false, true) { // did toggle?
		return nil
	}

	var retErr error
	for _, sps := range p.getPipelines().readers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := sps.Shutdown(ctx); err != nil {
			if retErr == nil {
				retErr = err
			} else {
				// Poor man's list of errors
				retErr = fmt.Errorf("%v; %v", retErr, err)
			}
		}
	}
	p.pipes.Store(&pipelines{})
	return retErr
}

func (p *TracerProvider) getPipelines() pipelines {
	return *(p.pipes.Load())
}

// TracerProviderOption configures a TracerProvider.
type TracerProviderOption interface {
	apply(tracerProviderConfig) tracerProviderConfig
}

type traceProviderOptionFunc func(tracerProviderConfig) tracerProviderConfig

func (fn traceProviderOptionFunc) apply(cfg tracerProviderConfig) tracerProviderConfig {
	return fn(cfg)
}

// WithSpanReader registers a SpanReader, which is the most general
// form of registration for a combination Sampler/Processor/Exporter.
func WithSpanReader(r SpanReader) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		if r != nil {
			cfg.readers = append(cfg.readers, r)
		}
		return cfg
	})
}

// WithSyncer registers the exporter with the TracerProvider using a
// SimpleSpanProcessor.
//
// This is not recommended for production use. The synchronous nature of the
// SimpleSpanProcessor that will wrap the exporter make it good for testing,
// debugging, or showing examples of other feature, but it will be slow and
// have a high computation resource usage overhead. The WithBatcher option is
// recommended for production use instead.
func WithSyncer(e SpanExporter) TracerProviderOption {
	return WithSpanProcessor(NewSimpleSpanProcessor(e))
}

// WithBatcher registers the exporter with the TracerProvider using a
// BatchSpanProcessor configured with the passed opts.
func WithBatcher(e SpanExporter, opts ...BatchSpanProcessorOption) TracerProviderOption {
	return WithSpanProcessor(NewBatchSpanProcessor(e, opts...))
}

// WithSpanProcessor registers the SpanProcessor with a TracerProvider.
func WithSpanProcessor(sp SpanProcessor) TracerProviderOption {
	return WithSpanReader(NewSimpleSpanReader(nil, sp))
}

// WithResource returns a TracerProviderOption that will configure the
// Resource r as a TracerProvider's Resource. The configured Resource is
// referenced by all the Tracers the TracerProvider creates. It represents the
// entity producing telemetry.
//
// If this option is not used, the TracerProvider will use the
// resource.Default() Resource by default.
func WithResource(r *resource.Resource) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		var err error
		cfg.resource, err = resource.Merge(resource.Environment(), r)
		if err != nil {
			otel.Handle(err)
		}
		return cfg
	})
}

// WithIDGenerator returns a TracerProviderOption that will configure the
// IDGenerator g as a TracerProvider's IDGenerator. The configured IDGenerator
// is used by the Tracers the TracerProvider creates to generate new Span and
// Trace IDs.
//
// If this option is not used, the TracerProvider will use a random number
// IDGenerator by default.
func WithIDGenerator(g IDGenerator) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		if g != nil {
			cfg.idGenerator = g
		}
		return cfg
	})
}

// WithSampler returns a TracerProviderOption that will configure the Sampler
// s as a TracerProvider's Sampler. The configured Sampler is used by the
// Tracers the TracerProvider creates to make their sampling decisions for the
// Spans they create.
//
// This option overrides the Sampler configured through the OTEL_TRACES_SAMPLER
// and OTEL_TRACES_SAMPLER_ARG environment variables. If this option is not used
// and the sampler is not configured through environment variables or the environment
// contains invalid/unsupported configuration, the TracerProvider will use a
// ParentBased(AlwaysSample) Sampler by default.
func WithSampler(s Sampler) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		if s != nil {
			cfg.sampler = s
		}
		return cfg
	})
}

// WithSpanLimits returns a TracerProviderOption that configures a
// TracerProvider to use the SpanLimits sl. These SpanLimits bound any Span
// created by a Tracer from the TracerProvider.
//
// If any field of sl is zero or negative it will be replaced with the default
// value for that field.
//
// If this or WithRawSpanLimits are not provided, the TracerProvider will use
// the limits defined by environment variables, or the defaults if unset.
// Refer to the NewSpanLimits documentation for information about this
// relationship.
//
// Deprecated: Use WithRawSpanLimits instead which allows setting unlimited
// and zero limits. This option will be kept until the next major version
// incremented release.
func WithSpanLimits(sl SpanLimits) TracerProviderOption {
	if sl.AttributeValueLengthLimit <= 0 {
		sl.AttributeValueLengthLimit = DefaultAttributeValueLengthLimit
	}
	if sl.AttributeCountLimit <= 0 {
		sl.AttributeCountLimit = DefaultAttributeCountLimit
	}
	if sl.EventCountLimit <= 0 {
		sl.EventCountLimit = DefaultEventCountLimit
	}
	if sl.AttributePerEventCountLimit <= 0 {
		sl.AttributePerEventCountLimit = DefaultAttributePerEventCountLimit
	}
	if sl.LinkCountLimit <= 0 {
		sl.LinkCountLimit = DefaultLinkCountLimit
	}
	if sl.AttributePerLinkCountLimit <= 0 {
		sl.AttributePerLinkCountLimit = DefaultAttributePerLinkCountLimit
	}
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		cfg.spanLimits = sl
		return cfg
	})
}

// WithRawSpanLimits returns a TracerProviderOption that configures a
// TracerProvider to use these limits. These limits bound any Span created by
// a Tracer from the TracerProvider.
//
// The limits will be used as-is. Zero or negative values will not be changed
// to the default value like WithSpanLimits does. Setting a limit to zero will
// effectively disable the related resource it limits and setting to a
// negative value will mean that resource is unlimited. Consequentially, this
// means that the zero-value SpanLimits will disable all span resources.
// Because of this, limits should be constructed using NewSpanLimits and
// updated accordingly.
//
// If this or WithSpanLimits are not provided, the TracerProvider will use the
// limits defined by environment variables, or the defaults if unset. Refer to
// the NewSpanLimits documentation for information about this relationship.
func WithRawSpanLimits(limits SpanLimits) TracerProviderOption {
	return traceProviderOptionFunc(func(cfg tracerProviderConfig) tracerProviderConfig {
		cfg.spanLimits = limits
		return cfg
	})
}

func applyTracerProviderEnvConfigs(cfg tracerProviderConfig) tracerProviderConfig {
	for _, opt := range tracerProviderOptionsFromEnv() {
		cfg = opt.apply(cfg)
	}

	return cfg
}

func tracerProviderOptionsFromEnv() []TracerProviderOption {
	var opts []TracerProviderOption

	sampler, err := samplerFromEnv()
	if err != nil {
		otel.Handle(err)
	}

	if sampler != nil {
		opts = append(opts, WithSampler(sampler))
	}

	return opts
}

// ensureValidTracerProviderConfig ensures that given TracerProviderConfig is valid.
func ensureValidTracerProviderConfig(cfg tracerProviderConfig) tracerProviderConfig {
	// @@@ Not sure what this default should be.
	// if cfg.sampler == nil {
	// 	cfg.sampler = ParentBased(AlwaysSample())
	// }
	if cfg.idGenerator == nil {
		cfg.idGenerator = defaultIDGenerator()
	}
	if cfg.resource == nil {
		cfg.resource = resource.Default()
	}
	return cfg
}
