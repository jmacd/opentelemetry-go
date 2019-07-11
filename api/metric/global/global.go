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

package global

import (
	"github.com/open-telemetry/opentelemetry-go/api/loader"
	"github.com/open-telemetry/opentelemetry-go/api/metric"
	"github.com/open-telemetry/opentelemetry-go/api/metric/internal"
)

func init() {
	if meter, _ := loader.Load().(metric.Meter); meter != nil {
		SetMeter(meter)
	}
}

// Meter return meter registered with global registry.
// If no meter is registered then an instance of noop Meter is returned.
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
