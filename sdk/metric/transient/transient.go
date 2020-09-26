// Copyright The OpenTelemetry Authors
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

package transient

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
)

type (
	Accumulator struct {
		standard    *sdk.Accumulator
		instruments sync.Map
	}

	syncImpl struct {
		standard  metric.SyncImpl
		refMapped sdk.RefcountMapped
	}

	boundSyncImpl struct {
		standard metric.BoundSyncImpl
		impl     *syncImpl
	}
)

var (
	ErrAsyncNotSupported     = fmt.Errorf("transient asynchronous instruments not supported")
	ErrInvalidBindInstrument = fmt.Errorf("invalid bind instrument")

	_ metric.MeterImpl     = &Accumulator{}
	_ metric.SyncImpl      = &syncImpl{}
	_ metric.BoundSyncImpl = &boundSyncImpl{}
)

func NewAccumulator(processor export.Processor, opts ...sdk.Option) *Accumulator {
	return &Accumulator{
		standard: sdk.NewAccumulator(processor, opts...),
	}
}

func (a *Accumulator) RecordBatch(ctx context.Context, labels []label.KeyValue, measurements ...metric.Measurement) {
	a.standard.RecordBatch(ctx, labels, measurements...)
}

func (a *Accumulator) NewSyncInstrument(descriptor metric.Descriptor) (metric.SyncImpl, error) {
	// TODO: key is instrumentation library, ... ?
	if actual, ok := a.instruments.Load(descriptor); ok {
		existing := actual.(*syncImpl)
		if existing.refMapped.Ref() {
			// the entry is in the map and will not be removed.
			return existing, nil
		}
		// This entry is no longer mapped, try to add a new entry.
	}

	s, err := a.standard.NewSyncInstrument(descriptor)
	if err != nil {
		return nil, err
	}

	impl := &syncImpl{
		standard:  s,
		refMapped: sdk.InitRefcountMapped(),
	}

	// TODO: the try-insert loop

	return impl, nil
}

func (a *Accumulator) NewAsyncInstrument(metric.Descriptor, metric.AsyncRunner) (metric.AsyncImpl, error) {
	return metric.NoopAsync{}, ErrAsyncNotSupported
}

func (a *Accumulator) Collect(ctx context.Context) int {
	return a.standard.Collect(ctx)
}

func (s *syncImpl) Implementation() interface{} {
	// TODO: comment
	return s.standard.Implementation()
}

func (s *syncImpl) Descriptor() metric.Descriptor {
	return s.standard.Descriptor()
}

func (s *syncImpl) Unref() {
	s.refMapped.Unref()
}

func (s *syncImpl) Bind(labels []label.KeyValue) metric.BoundSyncImpl {
	valid := s.refMapped.Ref()
	if !valid {
		panic(ErrInvalidBindInstrument)
	}
	return &boundSyncImpl{
		standard: s.standard.Bind(labels),
		impl:     s,
	}
}

func (s *syncImpl) RecordOne(ctx context.Context, number metric.Number, labels []label.KeyValue) {
	s.standard.RecordOne(ctx, number, labels)
}

func (b *boundSyncImpl) RecordOne(ctx context.Context, number metric.Number) {
	b.standard.RecordOne(ctx, number)
}

func (b *boundSyncImpl) Unbind() {
	b.standard.Unbind()
	b.impl.refMapped.Unref()
}
