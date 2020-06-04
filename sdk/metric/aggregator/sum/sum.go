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

package sum // import "go.opentelemetry.io/otel/sdk/metric/aggregator/sum"

import (
	"context"

	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator"
)

type sumValue struct {
	metric.Number
}

// Aggregator aggregates counter events.
type Aggregator struct {
	// current holds current increments to this counter record
	// current needs to be aligned for 64-bit atomic operations.
	current sumValue

	// checkpoint is a temporary used during Checkpoint()
	// checkpoint needs to be aligned for 64-bit atomic operations.
	checkpoint sumValue
}

var _ export.Aggregator = &Aggregator{}
var _ aggregation.Sum = &sumValue{}

// New returns a new counter aggregator implemented by atomic
// operations.  This aggregator implements the aggregation.Sum
// export interface.
func New() *Aggregator {
	return &Aggregator{}
}

// Checkpoint atomically saves the current value and resets the
// current sum to zero.
func (c *Aggregator) Checkpoint(_ *metric.Descriptor) {
	c.checkpoint.Number = c.current.Number.SwapNumberAtomic(metric.Number(0))
}

// Update atomically adds to the current value.
func (c *Aggregator) Update(_ context.Context, number metric.Number, desc *metric.Descriptor) error {
	c.current.AddNumberAtomic(desc.NumberKind(), number)
	return nil
}

// Merge combines two counters by adding their sums.
func (c *Aggregator) Merge(oa export.Aggregator, desc *metric.Descriptor) error {
	o, _ := oa.(*Aggregator)
	if o == nil {
		return aggregator.NewInconsistentMergeError(c, oa)
	}
	c.current.AddNumber(desc.NumberKind(), o.checkpoint.Number)
	return nil
}

func (c *Aggregator) Swap() {
	c.current, c.checkpoint = c.checkpoint, c.current
}

func (c *Aggregator) AccumulatedValue() aggregation.Aggregation {
	return &c.current
}

func (c *Aggregator) CheckpointedValue() aggregation.Aggregation {
	return &c.checkpoint
}

// Sum returns the last-checkpointed sum.  This will never return an
// error.
func (s *sumValue) Sum() (metric.Number, error) {
	return s.Number, nil
}

// Kind returns aggregation.SumKind.
func (s *sumValue) Kind() aggregation.Kind {
	return aggregation.SumKind
}
