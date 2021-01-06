// Copyright The OpenTelemetry Authors
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

package histogram // import "go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"

import (
	"context"
	"math/rand"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator"
)

// Note: This code uses a Mutex to govern access to the exclusive
// aggregator state.  This is in contrast to a lock-free approach
// (as in the Go prometheus client) that was reverted here:
// https://github.com/open-telemetry/opentelemetry-go/pull/669

type (
	// Aggregator observe events and counts them in pre-determined buckets.
	// It also calculates the sum and count of all events.
	Aggregator struct {
		lock           sync.Mutex
		boundaries     []float64
		numExemplars   int
		exemplarFilter ExemplarFilter
		kind           number.Kind
		state          *state
	}

	// ExemplarFilter
	ExemplarFilter func(ctx context.Context) bool

	// Config describes how the histogram is aggregated.
	Config struct {
		// ExplicitBoundaries support arbitrary bucketing schemes.  This
		// is the general case.
		ExplicitBoundaries []float64

		// ExemplarsPerBucket is the number of exemplars saved per bucket.
		ExemplarsPerBucket int

		// ExemplarFilter filters the set of updates used for
		// generating exemplars (e.g., to ensure they are
		// sampled for tracing).
		ExemplarFilter ExemplarFilter
	}

	// Option configures a histogram Config.
	Option interface {
		// Apply sets one or more Config fields.
		Apply(*Config)
	}

	// state represents the state of a histogram, consisting of
	// the sum and counts for all observed values and
	// the less than equal bucket count for the pre-determined boundaries.
	state struct {
		bucketCounts []uint64
		sum          number.Number
		count        uint64

		// bucketExemplars are stored as
		//   (b0, 0), ... (b0, numExemplars-1),
		//   (b1, 0), ... (b1, numExemplars-1),
		//   ...
		//   (bN, 0), ... (bN, numExemplars-1).
		//
		// Their context value may contain distributed context
		// details that will be expanded at the end of the
		// selection period.
		example *exampleState
	}

	exampleState struct {
		// exemplars are chosen by uniform random selection
		// (from those filtered, when filtering is configured).
		exemplars []aggregation.Example

		// filterCounts are the counts when exemplarFilter != nil.
		filterCounts []uint64
	}
)

// WithExplicitBoundaries sets the ExplicitBoundaries configuration option of a Config.
func WithExplicitBoundaries(explicitBoundaries []float64) Option {
	return explicitBoundariesOption{explicitBoundaries}
}

type explicitBoundariesOption struct {
	boundaries []float64
}

func (o explicitBoundariesOption) Apply(config *Config) {
	config.ExplicitBoundaries = o.boundaries
}

// WithExemplars sets the ExemplarsPerBucket configuration option of a Config.
func WithExemplars(numExemplars int) Option {
	return numExemplarsOption(numExemplars)
}

type numExemplarsOption int

func (o numExemplarsOption) Apply(config *Config) {
	config.ExemplarsPerBucket = int(o)
}

// WithExemplarFilter sets an optional filter used for exemplars, typically to
// ensure that exemplars are sampled (in case of sampling).
func WithExemplarFilter(filter ExemplarFilter) Option {
	return exemplarFilterOption(filter)
}

type exemplarFilterOption ExemplarFilter

func (o exemplarFilterOption) Apply(config *Config) {
	config.ExemplarFilter = ExemplarFilter(o)
}

// defaultExplicitBoundaries have been copied from prometheus.DefBuckets.
//
// Note we anticipate the use of a high-precision histogram sketch as
// the standard histogram aggregator for OTLP export.
// (https://github.com/open-telemetry/opentelemetry-specification/issues/982).
var defaultFloat64ExplicitBoundaries = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

// defaultInt64ExplicitBoundaryMultiplier determines the default
// integer histogram boundaries.
const defaultInt64ExplicitBoundaryMultiplier = 1e6

// defaultInt64ExplicitBoundaries applies a multiplier to the default
// float64 boundaries: [ 5K, 10K, 25K, ..., 2.5M, 5M, 10M ]
var defaultInt64ExplicitBoundaries = func(bounds []float64) (asint []float64) {
	for _, f := range bounds {
		asint = append(asint, defaultInt64ExplicitBoundaryMultiplier*f)
	}
	return
}(defaultFloat64ExplicitBoundaries)

var _ export.Aggregator = &Aggregator{}
var _ aggregation.Sum = &Aggregator{}
var _ aggregation.Count = &Aggregator{}
var _ aggregation.Histogram = &Aggregator{}
var _ aggregation.Exemplars = &Aggregator{}

// New returns a new aggregator for computing Histograms.
//
// A Histogram observe events and counts them in pre-defined buckets.
// And also provides the total sum and count of all observations.
//
// Note that this aggregator maintains each value using independent
// atomic operations, which introduces the possibility that
// checkpoints are inconsistent.
func New(cnt int, desc *metric.Descriptor, opts ...Option) []Aggregator {
	var cfg Config

	if desc.NumberKind() == number.Int64Kind {
		cfg.ExplicitBoundaries = defaultInt64ExplicitBoundaries
	} else {
		cfg.ExplicitBoundaries = defaultFloat64ExplicitBoundaries
	}

	for _, opt := range opts {
		opt.Apply(&cfg)
	}

	if cfg.ExemplarsPerBucket < 0 {
		cfg.ExemplarsPerBucket = 0
	}

	aggs := make([]Aggregator, cnt)

	// Boundaries MUST be ordered otherwise the histogram could not
	// be properly computed.
	sortedBoundaries := make([]float64, len(cfg.ExplicitBoundaries))

	copy(sortedBoundaries, cfg.ExplicitBoundaries)
	sort.Float64s(sortedBoundaries)

	for i := range aggs {
		aggs[i] = Aggregator{
			kind:         desc.NumberKind(),
			boundaries:   sortedBoundaries,
			numExemplars: cfg.ExemplarsPerBucket,
		}
		aggs[i].state = aggs[i].newState()
	}
	return aggs
}

// Aggregation returns an interface for reading the state of this aggregator.
func (c *Aggregator) Aggregation() aggregation.Aggregation {
	return c
}

// Kind returns aggregation.HistogramKind.
func (c *Aggregator) Kind() aggregation.Kind {
	return aggregation.HistogramKind
}

// Sum returns the sum of all values in the checkpoint.
func (c *Aggregator) Sum() (number.Number, error) {
	return c.state.sum, nil
}

// Count returns the number of values in the checkpoint.
func (c *Aggregator) Count() (uint64, error) {
	return c.state.count, nil
}

// Histogram returns the count of events in pre-determined buckets.
func (c *Aggregator) Histogram() (aggregation.Buckets, error) {
	return aggregation.Buckets{
		Boundaries: c.boundaries,
		Counts:     c.state.bucketCounts,
	}, nil
}

// SynchronizedMove saves the current state into oa and resets the current state to
// the empty set.  Since no locks are taken, there is a chance that
// the independent Sum, Count and Bucket Count are not consistent with each
// other.
func (c *Aggregator) SynchronizedMove(oa export.Aggregator, desc *metric.Descriptor) error {
	o, _ := oa.(*Aggregator)

	if oa != nil && o == nil {
		return aggregator.NewInconsistentAggregatorError(c, oa)
	}

	if o != nil {
		// Swap case: This is the ordinary case for a
		// synchronous instrument, where the SDK allocates two
		// Aggregators and lock contention is anticipated.
		// Reset the target state before swapping it under the
		// lock below.
		o.clearState()
	}

	c.lock.Lock()
	if o != nil {
		c.state, o.state = o.state, c.state
	} else {
		// No swap case: This is the ordinary case for an
		// asynchronous instrument, where the SDK allocates a
		// single Aggregator and there is no anticipated lock
		// contention.
		c.clearState()
	}
	c.lock.Unlock()

	return nil
}

func (c *Aggregator) newState() *state {
	var exs *exampleState
	numBuckets := len(c.boundaries) + 1
	if c.numExemplars != 0 {
		exs = &exampleState{
			exemplars: make([]aggregation.Example, c.numExemplars*numBuckets),
		}
		if c.exemplarFilter != nil {
			exs.filterCounts = make([]uint64, numBuckets)
		}
	}
	return &state{
		bucketCounts: make([]uint64, numBuckets),
		example:      exs,
	}
}

func (c *Aggregator) clearState() {
	for i := range c.state.bucketCounts {
		c.state.bucketCounts[i] = 0
	}

	if c.state.example != nil {
		for i := range c.state.example.exemplars {
			c.state.example.exemplars[i] = aggregation.Example{}
		}

		if c.state.example.filterCounts != nil {
			for i := range c.state.example.filterCounts {
				c.state.example.filterCounts[i] = 0
			}
		}
	}
	c.state.sum = 0
	c.state.count = 0
}

// Update adds the recorded measurement to the current data set.
func (c *Aggregator) Update(ctx context.Context, number number.Number, desc *metric.Descriptor) error {
	kind := desc.NumberKind()
	asFloat := number.CoerceToFloat64(kind)

	bucketID := len(c.boundaries)
	for i, boundary := range c.boundaries {
		if asFloat < boundary {
			bucketID = i
			break
		}
	}
	// Note: Binary-search was compared using the benchmarks. The following
	// code is equivalent to the linear search above:
	//
	//     bucketID := sort.Search(len(c.boundaries), func(i int) bool {
	//         return asFloat < c.boundaries[i]
	//     })
	//
	// The binary search wins for very large boundary sets, but
	// the linear search performs better up through arrays between
	// 256 and 512 elements, which is a relatively large histogram, so we
	// continue to prefer linear search.

	c.lock.Lock()
	defer c.lock.Unlock()

	c.state.count++
	c.state.sum.AddNumber(kind, number)
	c.state.bucketCounts[bucketID]++

	if c.numExemplars == 0 {
		return nil
	}

	var observed uint64

	if c.exemplarFilter == nil {
		observed = c.state.bucketCounts[bucketID]
	} else {
		if !c.exemplarFilter(ctx) {
			return nil
		}
		c.state.example.filterCounts[bucketID]++
		observed = c.state.example.filterCounts[bucketID]
	}

	base := c.numExemplars * bucketID

	if observed <= uint64(c.numExemplars) {
		c.state.example.exemplars[base+int(observed-1)] = aggregation.Example{
			Context: ctx,
			Number:  number,
			Time:    time.Now(),
		}
	} else {
		// TODO: avoid the global random number generator.
		// use per-Aggregator *rand.Rand?  use global pool?
		// Also it would be nice to move the random number
		// generation outside of the lock, but the calculation
		// with very small probability should repeat the
		// random number generation, ... the Int63n() call is
		// complex.
		index := rand.Int63n(int64(observed))

		if float64(index) < float64(c.numExemplars) {
			c.state.example.exemplars[base+int(index)] = aggregation.Example{
				Context: ctx,
				Number:  number,
				Time:    time.Now(),
			}
		}
	}
	return nil
}

// Merge combines two histograms that have the same buckets into a single one.
func (c *Aggregator) Merge(oa export.Aggregator, desc *metric.Descriptor) error {
	o, _ := oa.(*Aggregator)
	if o == nil {
		return aggregator.NewInconsistentAggregatorError(c, oa)
	}

	c.state.sum.AddNumber(desc.NumberKind(), o.state.sum)
	c.state.count += o.state.count

	for i := 0; i < len(c.state.bucketCounts); i++ {
		c.state.bucketCounts[i] += o.state.bucketCounts[i]
	}
	return nil
}

// ForEachExemplar implements aggregation.Exemplars
func (c *Aggregator) ForEachExemplar(f func(aggregation.Example)) error {
	if c.numExemplars == 0 {
		return nil
	}
	for bucket := range c.state.bucketCounts {
		var observed uint64
		if c.exemplarFilter == nil {
			observed = c.state.bucketCounts[bucket]
		} else {
			observed = c.state.example.filterCounts[bucket]
		}

		var have int
		if observed < uint64(c.numExemplars) {
			have = int(observed)
		} else {
			have = c.numExemplars
		}

		base := c.numExemplars * bucket
		for i := 0; i < have; i++ {
			f(c.state.example.exemplars[base+i])
		}
	}
	return nil
}
