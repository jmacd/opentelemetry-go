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

package label

import (
	"bytes"
	"sync"

	"go.opentelemetry.io/otel/api/core"
)

type Encoder interface {
	// Encode is called (concurrently) in instrumentation context.
	// It should return a unique representation of the labels
	// suitable for the SDK to use as a map key, an aggregator
	// grouping key, and/or the export encoding.
	Encode([]core.KeyValue) string
}

type defaultEncoder struct {
	// pool is a pool of labelset builders.  The buffers in this
	// pool grow to a size that most label encodings will not
	// allocate new memory.  This pool reduces the number of
	// allocations per new LabelSet to 3, typically, as seen in
	// the benchmarks.  (It should be 2--one for the LabelSet
	// object and one for the buffer.String() here--see the extra
	// allocation in the call to sort.Stable).
	pool sync.Pool // *bytes.Buffer
}

var _ Encoder = &defaultEncoder{}

func NewDefaultEncoder() Encoder {
	return &defaultEncoder{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

func (d *defaultEncoder) Encode(labels []core.KeyValue) string {
	buf := d.pool.Get().(*bytes.Buffer)
	defer d.pool.Put(buf)
	buf.Reset()

	for i, kv := range labels {
		if i > 0 {
			_, _ = buf.WriteRune(',')
		}
		_, _ = buf.WriteString(string(kv.Key))
		_, _ = buf.WriteRune('=')
		_, _ = buf.WriteString(kv.Value.Emit())
	}
	return buf.String()

}
