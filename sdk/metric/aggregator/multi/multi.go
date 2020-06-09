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

package multi // import "go.opentelemetry.io/otel/sdk/metric/aggregator/multi"

import (
	"context"

	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
)

type Aggregator struct {
	multiple   []export.Aggregator
	current    currentAggregation
	checkpoint checkpointAggregation
}

type checkpointAggregation struct {
	*Aggregator
}

type currentAggregation struct {
	*Aggregator
}

// TODO: improve error handling (compared w/ below)?

var _ export.Aggregator = &Aggregator{}
var _ aggregation.Multiple = &currentAggregation{}
var _ aggregation.Multiple = &checkpointAggregation{}

func New(aggs ...export.Aggregator) *Aggregator {
	a := &Aggregator{
		multiple: aggs,
	}
	a.current.Aggregator = a
	a.checkpoint.Aggregator = a
	return a
}

func (mu *Aggregator) Update(ctx context.Context, num metric.Number, desc *metric.Descriptor) error {
	var err error
	for _, agg := range mu.multiple {
		err1 := agg.Update(ctx, num, desc)
		if err1 != nil {
			err = err1
		}
	}
	return err
}

func (mu *Aggregator) Checkpoint(desc *metric.Descriptor) {
	for _, agg := range mu.multiple {
		agg.Checkpoint(desc)
	}
}

func (mu *Aggregator) Merge(agg export.Aggregator, desc *metric.Descriptor) error {
	var err error
	for _, agg := range mu.multiple {
		err1 := agg.Merge(agg, desc)
		if err1 != nil {
			err = err1
		}
	}
	return err
}

func (mu *Aggregator) Swap() {
	for _, agg := range mu.multiple {
		agg.Swap()
	}
}

func (mu *Aggregator) AccumulatedValue() aggregation.Aggregation {
	return &mu.current
}

func (mu *Aggregator) CheckpointedValue() aggregation.Aggregation {
	return &mu.checkpoint
}

func (*currentAggregation) Kind() aggregation.Kind {
	return aggregation.MultipleKind
}

func (*checkpointAggregation) Kind() aggregation.Kind {
	return aggregation.MultipleKind
}

func (ca *currentAggregation) Len() int {
	return len(ca.Aggregator.multiple)
}

func (ca *checkpointAggregation) Len() int {
	return len(ca.Aggregator.multiple)
}

func (ca *currentAggregation) Get(idx int) aggregation.Aggregation {
	return ca.Aggregator.multiple[idx].AccumulatedValue()
}

func (ca *checkpointAggregation) Get(idx int) aggregation.Aggregation {
	return ca.Aggregator.multiple[idx].CheckpointedValue()
}
