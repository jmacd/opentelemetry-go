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

import "sync"

// EmptyLabelSet is the empty label set.
var EmptyLabelSet LabelSet

// LabelEncoder enables an optimization for export pipelines that can
// pre-compute their label set encoding.
type LabelEncoder interface {
	// Encode is called (concurrently) in instrumentation context.
	// It should return a unique representation of the labels
	// suitable for the SDK to use as a map key, an aggregator
	// grouping key, and/or the export encoding.
	Encode([]KeyValue) LabelSet
}

// Labels stores complete information about a computed label set,
// including the labels in an appropriate order (as defined by the
// Batcher).  If the batcher does not re-order labels, they are
// presented in sorted order by the SDK.
type LabelSet struct {
	*labelSetImpl
}

const maxConcurrentEncoders = 3

type labelSetImpl struct {
	ordered []KeyValue

	lock     sync.Mutex
	encoders [maxConcurrentEncoders]LabelEncoder
	encoded  [maxConcurrentEncoders]string
}

// NewLabels builds a Labels object, consisting of an ordered set of
// labels, a unique encoded representation, and the encoder that
// produced it.
func NewLabels(ordered []KeyValue, encoded string, encoder LabelEncoder) LabelSet {
	lsi := &labelSetImpl{
		ordered: ordered,
	}
	// @@@
	// encoded: encoded,
	// encoder: encoder,
	return LabelSet{lsi}
}

// Ordered returns the labels in a specified order, according to the
// Batcher.
func (l LabelSet) Ordered() []KeyValue {
	return l.ordered
}

// Encoded is a pre-encoded form of the ordered labels.
func (l LabelSet) Encoded() string {
	return l.encoded
}

// Encoder is the encoder that computed the Encoded() representation.
func (l LabelSet) Encoder() LabelEncoder {
	return l.encoder
}

// Len returns the number of labels.
func (l LabelSet) Len() int {
	return len(l.ordered)
}

// // Labels returns a LabelSet corresponding to the arguments.  Passed
// // labels are de-duplicated, with last-value-wins semantics.
// func (m *SDK) Labels(kvs ...core.KeyValue) core.LabelSet {
// 	// Note: This computes a canonical encoding of the labels to
// 	// use as a map key.  It happens to use the encoding used by
// 	// statsd for labels, allowing an optimization for statsd
// 	// batchers.  This could be made configurable in the
// 	// constructor, to support the same optimization for different
// 	// batchers.

// 	// Check for empty set.
// 	if len(kvs) == 0 {
// 		return core.EmptyLabelSet
// 	}

// 	ls := core.labels{
// 		meter:  m,
// 		sorted: kvs,
// 	}

// 	// Sort and de-duplicate.
// 	sort.Stable(&ls.sorted)
// 	oi := 1
// 	for i := 1; i < len(ls.sorted); i++ {
// 		if ls.sorted[i-1].Key == ls.sorted[i].Key {
// 			ls.sorted[oi-1] = ls.sorted[i]
// 			continue
// 		}
// 		ls.sorted[oi] = ls.sorted[i]
// 		oi++
// 	}
// 	ls.sorted = ls.sorted[0:oi]

// 	ls.encoded = m.labelEncoder.Encode(ls.sorted)

// 	return ls
// }
