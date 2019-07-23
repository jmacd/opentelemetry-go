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

func New(res ...resource.Map) *SDK {
	// @@@
	return &SDK{}
}

// func (t *SDK) WithResources(attributes ...core.KeyValue) apitrace.Tracer {
// 	s := observer.NewScope(observer.ScopeID{
// 		EventID: t.resources,
// 	}, attributes...)
// 	return &tracer{
// 		resources: s.EventID,
// 	}
// }

// func (t *SDK) WithComponent(name string) apitrace.Tracer {
// 	return t.WithResources(ComponentKey.String(name))
// }

// func (t *SDK) WithService(name string) apitrace.Tracer {
// 	return t.WithResources(ServiceKey.String(name))
// }
