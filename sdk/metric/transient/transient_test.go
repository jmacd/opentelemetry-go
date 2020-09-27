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
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/processor/processortest"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestTransientDescriptors(t *testing.T) {
	ctx := context.Background()

	desc := metric.NewDescriptor(
		"counter.sum",
		metric.CounterKind,
		metric.Int64NumberKind,
		metric.WithDescription("a something counter"),
		metric.WithUnit("1"),
		metric.WithInstrumentationName("mylib"),
		metric.WithInstrumentationVersion("1.0"),
	)

	proc := processortest.NewProcessor(
		processortest.AggregatorSelector(),
		label.DefaultEncoder(),
	)
	acc := NewAccumulator(
		proc,
		sdk.WithResource(
			resource.New(label.String("R", "V")),
		),
	)
	labels := []label.KeyValue{
		label.String("A", "B"),
	}

	inst1, err := acc.NewSyncInstrument(desc)
	require.NoError(t, err)
	require.Equal(t, desc, inst1.Descriptor())

	inst2, err := acc.NewSyncInstrument(desc)
	require.NoError(t, err)
	require.Equal(t, desc, inst2.Descriptor())

	require.Equal(t, 1, acc.Size())

	inst1.RecordOne(ctx, metric.NewInt64Number(10), labels)
	inst2.RecordOne(ctx, metric.NewInt64Number(10), labels)
	inst2.Unref()

	firstCollected := acc.Collect(ctx)
	require.Equal(t, 1, firstCollected)
	require.Equal(t, 1, acc.Size())

	require.EqualValues(t, map[string]float64{
		"counter.sum/A=B/R=V": 20,
	}, proc.Values())

	inst1.RecordOne(ctx, metric.NewInt64Number(10), labels)
	inst1.Unref()

	secondCollected := acc.Collect(ctx)
	require.Equal(t, 1, secondCollected)
	require.Equal(t, 1, acc.Size())

	inst3, err := acc.NewSyncInstrument(desc)
	require.NoError(t, err)
	require.Equal(t, desc, inst3.Descriptor())

	inst3.RecordOne(ctx, metric.NewInt64Number(10), labels)
	inst3.Unref()

	thirdCollected := acc.Collect(ctx)
	require.Equal(t, 1, thirdCollected)
	require.Equal(t, 1, acc.Size())

	require.EqualValues(t, map[string]float64{
		"counter.sum/A=B/R=V": 40,
	}, proc.Values())

	fourthCollected := acc.Collect(ctx)
	require.Equal(t, 0, fourthCollected)
	require.Equal(t, 0, acc.Size())
}
