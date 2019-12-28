// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trace

import (
	"sync"
	"sync/atomic"

	export "go.opentelemetry.io/otel/sdk/export/trace"

	apitrace "go.opentelemetry.io/otel/api/trace"
)

const (
	defaultTracerName = "go.opentelemetry.io/otel/sdk/tracer"
)

// batcher contains export.SpanBatcher and its options.
type batcher struct {
	b    export.SpanBatcher
	opts []BatchSpanProcessorOption
}

// TracerOptions
type TracerOptions struct {
	syncers  []export.SpanSyncer
	batchers []batcher
	config   Config
}

type TracerOption func(*TracerOptions)

type Tracer struct {
	mu             sync.Mutex
	spanProcessors atomic.Value
	config         atomic.Value // access atomically
}

var _ apitrace.Tracer = &Tracer{}

// NewTracer creates an instance of trace provider. Optional
// parameter configures the provider with common options applicable
// to all tracer instances that will be created by this provider.
func NewTracer(opts ...TracerOption) (*Tracer, error) {
	o := &TracerOptions{}

	for _, opt := range opts {
		opt(o)
	}

	tp := &Tracer{}
	tp.config.Store(&Config{
		DefaultSampler:       ProbabilitySampler(defaultSamplingProbability),
		IDGenerator:          defIDGenerator(),
		MaxAttributesPerSpan: DefaultMaxAttributesPerSpan,
		MaxEventsPerSpan:     DefaultMaxEventsPerSpan,
		MaxLinksPerSpan:      DefaultMaxLinksPerSpan,
	})

	for _, syncer := range o.syncers {
		ssp := NewSimpleSpanProcessor(syncer)
		tp.RegisterSpanProcessor(ssp)
	}

	for _, batcher := range o.batchers {
		bsp, err := NewBatchSpanProcessor(batcher.b, batcher.opts...)
		if err != nil {
			return nil, err
		}
		tp.RegisterSpanProcessor(bsp)
	}

	tp.ApplyConfig(o.config)

	return tp, nil
}

// RegisterSpanProcessor adds the given SpanProcessor to the list of SpanProcessors
func (p *Tracer) RegisterSpanProcessor(s SpanProcessor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	new := make(spanProcessorMap)
	if old, ok := p.spanProcessors.Load().(spanProcessorMap); ok {
		for k, v := range old {
			new[k] = v
		}
	}
	new[s] = &sync.Once{}
	p.spanProcessors.Store(new)
}

// UnregisterSpanProcessor removes the given SpanProcessor from the list of SpanProcessors
func (p *Tracer) UnregisterSpanProcessor(s SpanProcessor) {
	mu.Lock()
	defer mu.Unlock()
	new := make(spanProcessorMap)
	if old, ok := p.spanProcessors.Load().(spanProcessorMap); ok {
		for k, v := range old {
			new[k] = v
		}
	}
	if stopOnce, ok := new[s]; ok && stopOnce != nil {
		stopOnce.Do(func() {
			s.Shutdown()
		})
	}
	delete(new, s)
	p.spanProcessors.Store(new)
}

// ApplyConfig changes the configuration of the provider.
// If a field in the configuration is empty or nil then its original value is preserved.
func (p *Tracer) ApplyConfig(cfg Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := *p.config.Load().(*Config)
	if cfg.DefaultSampler != nil {
		c.DefaultSampler = cfg.DefaultSampler
	}
	if cfg.IDGenerator != nil {
		c.IDGenerator = cfg.IDGenerator
	}
	if cfg.MaxEventsPerSpan > 0 {
		c.MaxEventsPerSpan = cfg.MaxEventsPerSpan
	}
	if cfg.MaxAttributesPerSpan > 0 {
		c.MaxAttributesPerSpan = cfg.MaxAttributesPerSpan
	}
	if cfg.MaxLinksPerSpan > 0 {
		c.MaxLinksPerSpan = cfg.MaxLinksPerSpan
	}
	p.config.Store(&c)
}

// WithSyncer options appends the syncer to the existing list of Syncers.
// This option can be used multiple times.
// The Syncers are wrapped into SimpleSpanProcessors and registered
// with the provider.
func WithSyncer(syncer export.SpanSyncer) TracerOption {
	return func(opts *TracerOptions) {
		opts.syncers = append(opts.syncers, syncer)
	}
}

// WithBatch options appends the batcher to the existing list of Batchers.
// This option can be used multiple times.
// The Batchers are wrapped into BatchedSpanProcessors and registered
// with the provider.
func WithBatcher(b export.SpanBatcher, bopts ...BatchSpanProcessorOption) TracerOption {
	return func(opts *TracerOptions) {
		opts.batchers = append(opts.batchers, batcher{b, bopts})
	}
}

// WithConfig option sets the configuration to provider.
func WithConfig(config Config) TracerOption {
	return func(opts *TracerOptions) {
		opts.config = config
	}
}
