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

package core

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"
)

// EmptyLabelSet is the empty label set.
var EmptyLabelSet LabelSet

type LabelEncoder interface {
	// Encode is called (concurrently) in instrumentation context.
	// It should return a unique representation of the labels
	// suitable for the SDK to use as a map key, an aggregator
	// grouping key, and/or the export encoding.
	Encode([]KeyValue) string
}

type LabelSet struct {
	*labelSetImpl
}

const maxConcurrentEncoders = 3

type sortedLabels []KeyValue

type labelSetImpl struct {
	ordered sortedLabels

	lock     sync.Mutex
	encoders [maxConcurrentEncoders]unsafe.Pointer
	encoded  [maxConcurrentEncoders]string
}

// Ordered returns the labels in a specified order, according to the
// Batcher.
func (l LabelSet) Ordered() []KeyValue {
	return l.ordered
}

// Encoded is a pre-encoded form of the ordered labels.
func (l LabelSet) Encoded(enc LabelEncoder) string {

	vptr := reflect.ValueOf(enc)
	if vptr.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Bad.. %T %v %v", vptr, vptr, vptr.Kind()))
	}
	search := unsafe.Pointer(vptr.Pointer())

	for i := 0; i < maxConcurrentEncoders; i++ {
		ptr := atomic.LoadPointer(&l.encoders[i])

		if ptr == search {
			return l.encoded[i]
		}

		if ptr == nil {
			break
		}
	}

	r := enc.Encode(l.ordered)

	l.lock.Lock()
	defer l.lock.Unlock()

	for i := 0; i < maxConcurrentEncoders; i++ {
		ptr := atomic.LoadPointer(&l.encoders[i])

		if ptr == search {
			break
		}

		if ptr == nil {
			break
		}
	}

	// TODO add a slice for overflow, test for panics

	return r
}

// Len returns the number of labels.
func (l LabelSet) Len() int {
	if l.labelSetImpl == nil {
		return 0
	}
	return len(l.ordered)
}

func (l LabelSet) Equals(o LabelSet) bool {
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

func (l LabelSet) AsMap() map[Key]Value {
	return map[Key]Value{}
}

// NewLabels builds a Labels object, consisting of an ordered set of
// labels, de-duplicated with last-value-wins semantics.
func NewLabels(kvs ...KeyValue) LabelSet {
	// Check for empty set.
	if len(kvs) == 0 {
		return EmptyLabelSet
	}

	ls := &labelSetImpl{
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

	return LabelSet{ls}
}

func (l *sortedLabels) Len() int {
	return len(*l)
}

func (l *sortedLabels) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

func (l *sortedLabels) Less(i, j int) bool {
	return (*l)[i].Key < (*l)[j].Key
}
