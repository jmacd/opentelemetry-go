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

package sdk

import (
	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/api/metric"
	"go.opentelemetry.io/api/resource"
	"go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/experimental/streaming/exporter/observer"
)

type SDK struct {
	resources observer.EventID
}

var _ trace.Tracer = &SDK{}
var _ metric.Meter = &SDK{}

func New(service string, resources ...resource.Map) *SDK {
	res := resource.Merge(append(resources, resource.Service(service))...)

	var all []core.KeyValue
	res.Foreach(func(kv core.KeyValue) bool {
		all = append(all, kv)
		return true
	})
	return &SDK{
		resources: observer.NewScope(observer.ScopeID{}, all...).EventID,
	}
}
