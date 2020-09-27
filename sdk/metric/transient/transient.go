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
	"runtime"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
)

type (
	Accumulator struct {
		standard    *sdk.Accumulator
		lock        sync.Mutex
		instruments sync.Map
	}

	syncImpl struct {
		standard  metric.SyncImpl
		refMapped sdk.RefcountMapped

		updateCount    int64
		collectedCount int64
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

func (a *Accumulator) NewSyncInstrument(descriptor metric.Descriptor) (metric.SyncImpl, error) {
	if actual, ok := a.instruments.Load(descriptor); ok {
		existing := actual.(*syncImpl)
		if existing.refMapped.Ref() {
			return existing, nil
		}
	}

	s, err := a.standard.NewSyncInstrument(descriptor)
	if err != nil {
		return nil, err
	}

	impl := &syncImpl{
		standard:  s,
		refMapped: sdk.InitRefcountMapped(),
	}

	for {
		if actual, loaded := a.instruments.LoadOrStore(descriptor, impl); loaded {
			existing := actual.(*syncImpl)
			if existing.refMapped.Ref() {
				return existing, nil
			}
			runtime.Gosched()
			continue
		}

		return impl, nil
	}
}

func (a *Accumulator) NewAsyncInstrument(metric.Descriptor, metric.AsyncRunner) (metric.AsyncImpl, error) {
	return metric.NoopAsync{}, ErrAsyncNotSupported
}

func (a *Accumulator) RecordBatch(ctx context.Context, labels []label.KeyValue, measurements ...metric.Measurement) {
	for _, m := range measurements {
		impl := m.SyncImpl().(*syncImpl)
		atomic.AddInt64(&impl.updateCount, 1)
	}
	a.standard.RecordBatch(ctx, labels, measurements...)
}

func (a *Accumulator) Collect(ctx context.Context) int {
	a.lock.Lock()
	defer a.lock.Unlock()
	defer a.purgeInstruments()
	return a.standard.Collect(ctx)
}

func (a *Accumulator) purgeInstruments() {
	a.instruments.Range(func(key, value interface{}) bool {
		impl := value.(*syncImpl)

		uses := atomic.LoadInt64(&impl.updateCount)
		coll := impl.collectedCount

		if uses != coll {
			impl.collectedCount = uses
			return true
		}

		if unmapped := impl.refMapped.TryUnmap(); !unmapped {
			return true
		}

		a.instruments.Delete(key)
		return true
	})
}

func (s *syncImpl) Implementation() interface{} {
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
	atomic.AddInt64(&s.updateCount, 1)
	s.standard.RecordOne(ctx, number, labels)
}

func (b *boundSyncImpl) RecordOne(ctx context.Context, number metric.Number) {
	atomic.AddInt64(&b.impl.updateCount, 1)
	b.standard.RecordOne(ctx, number)
}

func (b *boundSyncImpl) Unbind() {
	b.standard.Unbind()
	b.impl.refMapped.Unref()
}

func (a *Accumulator) Size() int {
	sz := 0
	a.instruments.Range(func(_, _ interface{}) bool {
		sz++
		return true
	})
	return sz
}
