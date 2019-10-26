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

package counter

import (
	"context"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/sdk/export"
)

type (
	// Aggregator aggregates counter events.
	Aggregator struct {
		// live holds current increments to this counter record
		live core.Number

		// save is a temporary used during Collect()
		save core.Number
	}
)

var _ export.MetricAggregator = &Aggregator{}

// New returns a new counter aggregator.  This aggregator computes an
// atomic sum.
func New() *Aggregator {
	return &Aggregator{}
}

// AsInt64 returns the accumulated count as an int64.
func (c *Aggregator) AsNumber() core.Number {
	return c.save.AsNumber()
}

// Collect saves the current value (atomically) and exports it.
func (c *Aggregator) Collect(ctx context.Context, rec export.MetricRecord, exp export.MetricBatcher) {
	desc := rec.Descriptor()
	kind := desc.NumberKind()
	zero := core.Number(0)
	c.save = c.live.SwapNumberAtomic(zero)

	if c.save.IsZero(kind) {
		return
	}

	exp.Export(ctx, rec, c)
}

// Collect updates the current value (atomically) for later export.
func (c *Aggregator) Update(_ context.Context, number core.Number, rec export.MetricRecord) {
	desc := rec.Descriptor()
	kind := desc.NumberKind()
	if !desc.Alternate() && number.IsNegative(kind) {
		// TODO warn
		return
	}

	c.live.AddNumberAtomic(kind, number)
}

func (c *Aggregator) Merge(oa export.MetricAggregator, desc *export.Descriptor) {
	o, _ := oa.(*Aggregator)
	if o == nil {
		// TODO warn
		return
	}
	c.save.AddNumber(desc.NumberKind(), o.save)
}
