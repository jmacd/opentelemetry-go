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

	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/api/metric"
	"go.opentelemetry.io/api/metric/internal"
)

// Meter returns the global Meter instance.  Before opentelemetry.Init() is
// called, this returns an "indirect" No-op implementation, that will be updated
// once the SDK is initialized.
func Meter() metric.Meter {
	if t := internal.Global.Load(); t != nil {
		return t.(metric.Meter)
	}
	return metric.NoopMeter{}
}

// SetMeter sets provided meter as a global meter.
func SetMeter(t metric.Meter) {
	internal.Global.Store(t)
}

// IndirectMeter implements Meter, allows callers of Meter() before Init()
// to forward to the installed SDK.
type IndirectMeter struct{}

var globalIndirect metric.Meter = &IndirectMeter{}

// GetFloat64Gauge implements Meter
func (*IndirectMeter) GetFloat64Gauge(
	ctx context.Context,
	gauge *metric.Float64GaugeHandle,
	labels ...core.KeyValue,
) metric.Float64Gauge {
	return Meter().GetFloat64Gauge(ctx, gauge, labels...)
}
