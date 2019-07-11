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
	"github.com/open-telemetry/opentelemetry-go/api/stats"
	"github.com/open-telemetry/opentelemetry-go/api/stats/internal"
)

func init() {
	if recorder, _ := loader.Load().(stats.Recorder); recorder != nil {
		SetRecorder(recorder)
	}
}

// Recorder return meter registered with global registry.
// If no meter is registered then an instance of noop Recorder is returned.
func Recorder() stats.Recorder {
	if t := internal.Global.Load(); t != nil {
		return t.(stats.Recorder)
	}
	return stats.NoopRecorder{}
}

// SetRecorder sets provided meter as a global meter.
func SetRecorder(t stats.Recorder) {
	internal.Global.Store(t)
}
