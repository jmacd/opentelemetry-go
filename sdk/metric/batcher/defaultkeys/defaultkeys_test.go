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

package defaultkeys_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/core"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/batcher/defaultkeys"
	"go.opentelemetry.io/otel/sdk/metric/batcher/test"
)

func TestGroupingStateless(t *testing.T) {
	ctx := context.Background()
	enc := sdk.NewDefaultLabelEncoder()
	b := defaultkeys.New(test.NewAggregationSelector(), enc, false)

	_ = b.Process(ctx, test.NewGaugeRecord(test.GaugeADesc, test.Labels1, 10))
	_ = b.Process(ctx, test.NewGaugeRecord(test.GaugeADesc, test.Labels2, 20))
	_ = b.Process(ctx, test.NewGaugeRecord(test.GaugeADesc, test.Labels3, 30))

	_ = b.Process(ctx, test.NewGaugeRecord(test.GaugeBDesc, test.Labels1, 10))
	_ = b.Process(ctx, test.NewGaugeRecord(test.GaugeBDesc, test.Labels2, 20))
	_ = b.Process(ctx, test.NewGaugeRecord(test.GaugeBDesc, test.Labels3, 30))

	_ = b.Process(ctx, test.NewCounterRecord(test.CounterADesc, test.Labels1, 10))
	_ = b.Process(ctx, test.NewCounterRecord(test.CounterADesc, test.Labels2, 20))
	_ = b.Process(ctx, test.NewCounterRecord(test.CounterADesc, test.Labels3, 40))

	_ = b.Process(ctx, test.NewCounterRecord(test.CounterBDesc, test.Labels1, 10))
	_ = b.Process(ctx, test.NewCounterRecord(test.CounterBDesc, test.Labels2, 20))
	_ = b.Process(ctx, test.NewCounterRecord(test.CounterBDesc, test.Labels3, 40))

	checkpointSet := b.CheckpointSet()
	b.FinishedCollection()

	output := test.NewOutput(enc)
	checkpointSet.ForEach(output.AddTo)

	// Repeat for {counter,gauge}.{1,2}.
	// Output gauge should have only the "G=H" and "G=" keys.
	// Output counter should have only the "C=D" and "C=" keys.
	require.EqualValues(t, map[string]int64{
		"counter.a/C=D": 30, // labels1 + labels2
		"counter.a/C=":  40, // labels3
		"counter.b/C=D": 30, // labels1 + labels2
		"counter.b/C=":  40, // labels3
		"gauge.a/G=H":   10, // labels1
		"gauge.a/G=":    30, // labels3 = last value
		"gauge.b/G=H":   10, // labels1
		"gauge.b/G=":    30, // labels3 = last value
	}, output.Values)

	// Verify that state is reset by FinishedCollection()
	checkpointSet = b.CheckpointSet()
	b.FinishedCollection()
	checkpointSet.ForEach(func(rec export.Record) {
		t.Fatal("Unexpected call")
	})
}

func TestGroupingStateful(t *testing.T) {
	ctx := context.Background()
	b := defaultkeys.New(test.NewAggregationSelector(), test.GroupEncoder, true)

	counterA := test.NewCounterRecord(test.CounterADesc, test.Labels1, 10)
	caggA := counterA.Aggregator()
	_ = b.Process(ctx, counterA)

	counterB := test.NewCounterRecord(test.CounterBDesc, test.Labels1, 10)
	caggB := counterB.Aggregator()
	_ = b.Process(ctx, counterB)

	checkpointSet := b.CheckpointSet()
	b.FinishedCollection()

	outEnc := sdk.NewDefaultLabelEncoder()
	output1 := test.NewOutput(outEnc)
	checkpointSet.ForEach(output1.AddTo)

	require.EqualValues(t, map[string]int64{
		"counter.a/C=D": 10, // labels1
		"counter.b/C=D": 10, // labels1
	}, output1.Values)

	// Test that state was NOT reset
	checkpointSet = b.CheckpointSet()
	b.FinishedCollection()

	output2 := test.NewOutput(outEnc)
	checkpointSet.ForEach(output2.AddTo)

	require.EqualValues(t, output1, output2)

	// Update and re-checkpoint the original record.
	_ = caggA.Update(ctx, core.NewInt64Number(20), test.CounterADesc)
	_ = caggB.Update(ctx, core.NewInt64Number(20), test.CounterBDesc)
	caggA.Checkpoint(ctx, test.CounterADesc)
	caggB.Checkpoint(ctx, test.CounterBDesc)

	// As yet cagg has not been passed to Batcher.Process.  Should
	// not see an update.
	checkpointSet = b.CheckpointSet()
	b.FinishedCollection()

	output3 := test.NewOutput(outEnc)
	checkpointSet.ForEach(output3.AddTo)

	require.EqualValues(t, output1, output3)

	// Now process the second update
	_ = b.Process(ctx, export.NewRecord(test.CounterADesc, test.Labels1, caggA))
	_ = b.Process(ctx, export.NewRecord(test.CounterBDesc, test.Labels1, caggB))

	checkpointSet = b.CheckpointSet()
	b.FinishedCollection()

	output4 := test.NewOutput(outEnc)
	checkpointSet.ForEach(output4.AddTo)

	require.EqualValues(t, map[string]int64{
		"counter.a/C=D": 30,
		"counter.b/C=D": 30,
	}, output4.Values)
}
