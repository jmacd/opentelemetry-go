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

package resource

import (
	"go.opentelemetry.io/api/core"
	"go.opentelemetry.io/api/key"
	"go.opentelemetry.io/api/tag"
)

var (
	ComponentKey = key.New("component")
	ServiceKey   = key.New("service")
)

type Map struct {
	labels tag.Map
}

func Service(name string) Map {
	return Map{tag.NewMap(tag.MapUpdate{SingleKV: ServiceKey.String(name)})}
}

func Component(name string) Map {
	return Map{tag.NewMap(tag.MapUpdate{SingleKV: ComponentKey.String(name)})}
}

func New(labels ...core.KeyValue) Map {
	return Map{tag.NewMap(tag.MapUpdate{MultiKV: labels})}
}

func Merge(maps ...Map) Map {
	var all []core.KeyValue
	for _, m := range maps {
		m.labels.Foreach(func(kv core.KeyValue) bool {
			all = append(all, kv)
			return true
		})
	}
	return Map{tag.NewMap(tag.MapUpdate{MultiKV: all})}
}
