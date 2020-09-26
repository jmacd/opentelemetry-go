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

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
)

type Accumulator struct {
	standard *sdk.Accumulator
}

var (
	ErrAsyncNotSupported = fmt.Errorf("transient asynchronous instruments not supported")

	_ metric.MeterImpl = &Accumulator{}
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
	return a.standard.NewSyncInstrument(descriptor)
}

func (a *Accumulator) NewAsyncInstrument(
	descriptor metric.Descriptor,
	runner metric.AsyncRunner,
) (metric.AsyncImpl, error) {
	return metric.NoopAsync{}, ErrAsyncNotSupported
}

func (m *Accumulator) Collect(ctx context.Context) int {
	return 0
}
