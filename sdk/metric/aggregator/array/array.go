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

package array

import (
	"context"
	"sort"
	"sync"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/sdk/export"
)

type (
	Aggregator struct {
		lock   sync.Mutex
		points Points
		saved  Points
		sum    core.Number
	}

	Points []core.Number
)

var _ export.MetricAggregator = &Aggregator{}

func New() *Aggregator {
	return &Aggregator{}
}

func (a *Aggregator) Collect(ctx context.Context, rec export.MetricRecord, exp export.MetricBatcher) {
	a.lock.Lock()
	a.saved, a.points = a.points, nil
	a.lock.Unlock()

	if a.saved == nil {
		return
	}

	sort.Sort(&a.saved)

	a.sum = core.Number(0)
	desc := rec.Descriptor()

	for _, v := range a.saved {
		a.sum.AddNumber(desc.NumberKind(), v)
	}

	exp.Export(ctx, rec, a)
}

func (a *Aggregator) Update(_ context.Context, number core.Number, rec export.MetricRecord) {

	desc := rec.Descriptor()
	kind := desc.NumberKind()

	if !desc.Alternate() && number.IsNegative(kind) {
		// TODO warn
		return
	}

	a.lock.Lock()
	a.points = append(a.points, number)
	a.lock.Unlock()
}

func (a *Aggregator) Merge(oa export.MetricAggregator, _ *export.Descriptor) {
	o, _ := oa.(*Aggregator)
	if o == nil {
		// TODO warn
		return
	}
	// Called in a single threaded context, no locks needed.
	a.points = append(a.points, o.points...)
}

func (p *Points) Len() int {
	return len(*p)
}

func (p *Points) Less(i, j int) bool {
	return (*p)[i] < (*p)[j]
}

func (p *Points) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}
