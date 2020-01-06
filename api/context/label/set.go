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
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"go.opentelemetry.io/otel/api/core"
)

// EmptySet is the empty label set.
var EmptySet = Set{
	&setImpl{},
}

type Set struct {
	*setImpl
}

const maxConcurrentEncoders = 3

type sorted []core.KeyValue

type setImpl struct {
	ordered sorted

	lock     sync.Mutex
	encoders [maxConcurrentEncoders]unsafe.Pointer
	encoded  [maxConcurrentEncoders]string
}

// Ordered returns the labels in a specified order, according to the
// Batcher.
func (l Set) Ordered() []core.KeyValue {
	if l.setImpl == nil {
		return nil
	}
	return l.ordered
}

func (l Set) String() string {
	return fmt.Sprint(l.Ordered())
}

func (l Set) Value(k core.Key) (core.Value, bool) {
	if l.setImpl == nil {
		return core.Value{}, false
	}
	idx := sort.Search(len(l.ordered), func(i int) bool {
		return l.ordered[i].Key >= k
	})
	if idx < len(l.ordered) && k == l.ordered[idx].Key {
		return l.ordered[idx].Value, true
	}
	return core.Value{}, false
}

func (l Set) HasValue(k core.Key) bool {
	_, ok := l.Value(k)
	return ok
}

// Encoded is a pre-encoded form of the ordered labels.
func (l Set) Encoded(enc Encoder) string {
	if enc == nil {
		return ""
	}

	vptr := reflect.ValueOf(enc)
	if vptr.Kind() != reflect.Ptr {
		panic("Encoder implementations must use pointer receivers")
	}
	myself := unsafe.Pointer(vptr.Pointer())

	idx := 0
	for idx := 0; idx < maxConcurrentEncoders; idx++ {
		ptr := atomic.LoadPointer(&l.encoders[idx])

		if ptr == myself {
			// fmt.Println("Case A")
			return l.encoded[idx]
		}

		if ptr == nil {
			// fmt.Println("Case B", idx)
			break
		}
	}

	r := enc.Encode(l.ordered)

	l.lock.Lock()
	defer l.lock.Unlock()

	for ; idx < maxConcurrentEncoders; idx++ {
		ptr := atomic.LoadPointer(&l.encoders[idx])

		if ptr != nil {
			// fmt.Println("Case C")
			continue
		}

		if ptr == nil {
			// fmt.Println("Case D", idx)
			atomic.StorePointer(&l.encoders[idx], myself)
			l.encoded[idx] = r
			break
		}
	}

	// TODO add a slice for overflow, test for panics

	return r
}

// Len returns the number of labels.
func (l Set) Len() int {
	if l.setImpl == nil {
		return 0
	}
	return len(l.ordered)
}

func (l Set) Equals(o Set) bool {
	if l.Len() != o.Len() {
		return false
	}
	for i := 0; i < l.Len(); i++ {
		if l.ordered[i] != o.ordered[i] {
			return false
		}
	}
	return true
}

func (l Set) AsMap() map[core.Key]core.Value {
	r := map[core.Key]core.Value{}
	for _, kv := range l.ordered {
		r[kv.Key] = kv.Value
	}
	return r
}

// NewSet builds a Labels object, consisting of an ordered set of
// labels, de-duplicated with last-value-wins semantics.
func NewSet(kvs ...core.KeyValue) Set {
	// Check for empty set.
	if len(kvs) == 0 {
		return EmptySet
	}

	ls := &setImpl{
		ordered: kvs,
	}

	// Sort and de-duplicate.
	sort.Stable(&ls.ordered)
	oi := 1
	for i := 1; i < len(ls.ordered); i++ {
		if ls.ordered[i-1].Key == ls.ordered[i].Key {
			ls.ordered[oi-1] = ls.ordered[i]
			continue
		}
		ls.ordered[oi] = ls.ordered[i]
		oi++
	}
	ls.ordered = ls.ordered[0:oi]

	return Set{ls}
}

func (l *sorted) Len() int {
	return len(*l)
}

func (l *sorted) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

func (l *sorted) Less(i, j int) bool {
	return (*l)[i].Key < (*l)[j].Key
}
