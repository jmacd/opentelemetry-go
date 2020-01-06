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
	"go.opentelemetry.io/otel/api/context/label"
	"go.opentelemetry.io/otel/api/core"
)

// TODO Comments needed! This was formerly known as distributedcontext.Map

type Map struct {
	ls label.Set
}

type MapUpdate struct {
	SingleKV core.KeyValue
	MultiKV  []core.KeyValue
}

func Empty() Map {
	return Map{}
}

func New(update MapUpdate) Map {
	return Empty().Apply(update)
}

func (m Map) Apply(update MapUpdate) Map {
	one := 0
	if update.SingleKV.Key.Defined() {
		one = 1
	}

	ls := make([]core.KeyValue, 0, m.Len()+len(update.MultiKV)+one)
	ls = append(ls, m.ls.Ordered()...)
	if one == 1 {
		ls = append(ls, update.SingleKV)
	}

	ls = append(ls, update.MultiKV...)

	return Map{ls: label.NewSet(ls...)}
}

func (m Map) Value(k core.Key) (core.Value, bool) {
	return m.ls.Value(k)
}

func (m Map) HasValue(k core.Key) bool {
	return m.ls.HasValue(k)
}

func (m Map) Len() int {
	return m.ls.Len()
}

func (m Map) Ordered() []core.KeyValue {
	return m.ls.Ordered()
}

func (m Map) Foreach(f func(kv core.KeyValue) bool) {
	for _, kv := range m.ls.Ordered() {
		if !f(kv) {
			return
		}
	}
}
