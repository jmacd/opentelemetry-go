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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
)

func TestMap(t *testing.T) {
	for _, testcase := range []struct {
		name       string
		updateOne  core.KeyValue
		updateMany []core.KeyValue
		init       []int
		wantKVs    []core.KeyValue
	}{
		{
			name: "NewMap with MultiKV",
			updateMany: []core.KeyValue{
				key.Int64("key1", 1),
				key.String("key2", "val2"),
			},
			init: []int{},
			wantKVs: []core.KeyValue{
				key.Int64("key1", 1),
				key.String("key2", "val2"),
			},
		},
		{
			name:      "NewMap with SingleKV",
			updateOne: key.String("key1", "val1"),
			init:      []int{},
			wantKVs: []core.KeyValue{
				key.String("key1", "val1"),
			},
		},
		{
			name:    "NewMap with empty MapUpdate",
			init:    []int{},
			wantKVs: nil,
		},
		{
			name: "Map with MultiKV",
			updateMany: []core.KeyValue{
				key.Int64("key1", 1),
				key.String("key2", "val2"),
			},
			init: []int{5},
			wantKVs: []core.KeyValue{
				key.Int64("key1", 1),
				key.String("key2", "val2"),
				key.Int("key5", 5),
			},
		},
		{
			name:      "Map with SingleKV",
			updateOne: key.String("key1", "val1"),
			init:      []int{5},
			wantKVs: []core.KeyValue{
				key.String("key1", "val1"),
				key.Int("key5", 5),
			},
		},
		{
			name:       "Map with empty MapUpdate",
			updateMany: []core.KeyValue{},
			init:       []int{5},
			wantKVs: []core.KeyValue{
				key.Int("key5", 5),
			},
		},
	} {
		t.Logf("Running test case %s", testcase.name)
		got := Empty()
		if len(testcase.init) > 0 {
			got = makeTestMap(testcase.init)
		}
		if testcase.updateMany != nil {
			got = got.AddMany(testcase.updateMany...)
		}
		if testcase.updateOne.Key.Defined() {
			got = got.AddOne(testcase.updateOne)
		}
		for _, s := range testcase.wantKVs {
			if ok := got.HasValue(s.Key); !ok {
				t.Errorf("Expected Key %s to have Value", s.Key)
			}
			if g, ok := got.Value(s.Key); !ok || g != s.Value {
				t.Errorf("+got: %v, -want: %v", g, s.Value)
			}
		}
		// test Foreach()
		got.Foreach(func(kv core.KeyValue) bool {
			for _, want := range testcase.wantKVs {
				if kv == want {
					return false
				}
			}
			t.Errorf("Expected kv %v, but not found", kv)
			return true
		})
		require.EqualValues(t, got.Ordered(), testcase.wantKVs)
	}
}

func makeTestMap(ints []int) Map {
	var r []core.KeyValue
	for _, v := range ints {
		r = append(r, core.Key(fmt.Sprintf("key%d", v)).Int(v))
	}
	return New(r...)
}
