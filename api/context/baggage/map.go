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

package baggage

import (
	"go.opentelemetry.io/otel/api/context/internal"
	"go.opentelemetry.io/otel/api/core"
)

type Map struct {
	set *internal.Set
}

func Empty() Map {
	return Map{internal.EmptySet()}
}

func New(kvs ...core.KeyValue) Map {
	return Empty().AddMany(kvs...)
}

func (m Map) AddOne(kv core.KeyValue) Map {
	return Map{m.set.AddOne(kv)}
}

func (m Map) AddMany(kvs ...core.KeyValue) Map {
	return Map{m.set.AddMany(kvs...)}
}

func (m Map) Value(k core.Key) (core.Value, bool) {
	return m.set.Value(k)
}

func (m Map) HasValue(k core.Key) bool {
	return m.set.HasValue(k)
}

func (m Map) Len() int {
	return m.set.Len()
}

func (m Map) Ordered() []core.KeyValue {
	return m.set.Ordered()
}

func (m Map) Foreach(f func(kv core.KeyValue) bool) {
	for _, kv := range m.set.Ordered() {
		if !f(kv) {
			return
		}
	}
}
