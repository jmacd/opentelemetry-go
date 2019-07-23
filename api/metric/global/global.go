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

package metric

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/api/metric"
)

// The global indirect meter is the default value
var global = &IndirectMeter{}

// IndirectMeter implements Meter
type IndirectMeter struct {
	meter atomic.Value
}

var _ metric.Meter = &IndirectMeter{}

var noopMeter = &metric.NoopMeter{}

// Meter return meter registered with global registry.
// If no meter is registered then an instance of noop Meter is returned.
func Meter() metric.Meter {
	return global
}

// SetMeter sets provided meter as a global meter.
func SetMeter(t metric.Meter) {
	global.meter.Store(t)
}

// getMeter gets the current global meter or a noop
func (im *IndirectMeter) getMeter() metric.Meter {
	if t := im.meter.Load(); t != nil {
		return t.(metric.Meter)
	}
	return noopMeter
}

// GetFloat64Gauge implements metric.Meter
func (im *IndirectMeter) GetFloat64Gauge(
	ctx context.Context,
	gauge *metric.Float64GaugeHandle,
	labels ...core.KeyValue,
) metric.Float64Gauge {
	return im.getMeter().GetFloat64Gauge(ctx, gauge, labels...)
}
