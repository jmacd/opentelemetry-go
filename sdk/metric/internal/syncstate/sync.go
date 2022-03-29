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

package syncstate

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"

	"go.opentelemetry.io/otel/sdk/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/metric/internal/viewstate"
	"go.opentelemetry.io/otel/sdk/metric/number"
	"go.opentelemetry.io/otel/sdk/metric/sdkapi"
)

// Performance note: there is still 1 obligatory allocation in the
// fast path of this code due to the sync.Map key.  Assuming Go will
// give us a generic form of sync.Map some time soon, the allocation
// cost of instrument.Current will be reduced to zero allocs in the
// fast path.  See also https://github.com/a8m/syncmap.

type (
	Instrument struct {
		instrument.Synchronous

		descriptor sdkapi.Descriptor
		compiled   viewstate.Instrument
		current    sync.Map // map[attribute.Set]*record
	}

	record struct {
		instrument.Synchronous

		refMapped  refcountMapped
		instrument *Instrument

		// updateCount is incremented on every Update.
		updateCount int64

		// collectedCount is set to updateCount on collection,
		// supports checking for no updates during a round.
		collectedCount int64

		distinct    attribute.Set
		attributes  []attribute.KeyValue
		accumulator viewstate.Accumulator
	}

	counter[N number.Any] struct {
		*Instrument
	}

	histogram[N number.Any] struct {
		*Instrument
	}
)

var (
	_ syncint64.Counter         = counter[int64]{}
	_ syncint64.UpDownCounter   = counter[int64]{}
	_ syncint64.Histogram       = histogram[int64]{}
	_ syncfloat64.Counter       = counter[float64]{}
	_ syncfloat64.UpDownCounter = counter[float64]{}
	_ syncfloat64.Histogram     = histogram[float64]{}
)

func NewInstrument(desc sdkapi.Descriptor, compiled viewstate.Instrument) *Instrument {
	return &Instrument{
		descriptor: desc,
		compiled:   compiled,
	}
}

func (inst *Instrument) Descriptor() sdkapi.Descriptor {
	return inst.descriptor
}

func NewCounter[N number.Any](inst *Instrument) counter[N] {
	return counter[N]{Instrument: inst}
}

func NewHistogram[N number.Any](inst *Instrument) histogram[N] {
	return histogram[N]{Instrument: inst}
}

func (c counter[N]) Add(ctx context.Context, incr N, attrs ...attribute.KeyValue) {
	if c.Instrument != nil {
		capture(ctx, c.Instrument, incr, attrs)
	}
}

func (h histogram[N]) Record(ctx context.Context, incr N, attrs ...attribute.KeyValue) {
	if h.Instrument != nil {
		capture(ctx, h.Instrument, incr, attrs)
	}
}

// func (a *Provider) Collect(r *reader.Reader, sequence viewstate.Sequence, output *[]reader.Instrument) {
// 	a.instrumentsLock.Lock()
// 	instruments := a.instruments
// 	a.instrumentsLock.Unlock()

// 	*output = make([]reader.Instrument, len(instruments))

// 	for instIdx, inst := range instruments {
// 		iout := &(*output)[instIdx]

// 		iout.Instrument = inst.descriptor

// 		// Note: @@@ Feels like this should be set in a
// 		// compiled class of the viewstate package, selected
// 		// when the instrument is constructed, that knows its
// 		// temporality.
// 		_, iout.Temporality = r.Defaults()(inst.descriptor.InstrumentKind())

// 		inst.compiled.PrepareCollect(r, sequence)

// 		inst.current.Range(func(key interface{}, value interface{}) bool {
// 			rec := value.(*record)
// 			any := a.collectRecord(rec, false)

// 			if any != 0 {
// 				return true
// 			}
// 			// Having no updates since last collection, try to unmap:
// 			if unmapped := rec.refMapped.tryUnmap(); !unmapped {
// 				// The record is referenced by a binding, continue.
// 				return true
// 			}

// 			// If any other goroutines are now trying to re-insert this
// 			// entry in the map, they are busy calling Gosched() awaiting
// 			// this deletion:
// 			inst.current.Delete(key)

// 			// Last we'll see of this.
// 			_ = a.collectRecord(rec, true)
// 			return true
// 		})
// 		inst.compiled.Collect(r, sequence, &iout.Series)
// 	}
// }

// func (a *Provider) collectRecord(rec *record, final bool) int {
// 	mods := atomic.LoadInt64(&rec.updateCount)
// 	coll := rec.collectedCount

// 	if mods == coll {
// 		return 0
// 	}
// 	// Updates happened in this interval,
// 	// collect and continue.
// 	rec.collectedCount = mods

// 	// Note: We could use the `final` bit here to signal to the
// 	// receiver of this aggregation that it is the last in a
// 	// sequence and it should feel encouraged to forget its state
// 	// because a new accumulator will be built to continue this
// 	// stream (w/ a new *record).
// 	_ = final

// 	if rec.accumulator == nil {
// 		return 0
// 	}
// 	rec.accumulator.Accumulate()
// 	return 1
// }

func capture[N number.Any](_ context.Context, inst *Instrument, num N, attrs []attribute.KeyValue) {
	// TODO: Here, this is the place to use context, extract baggage.

	rec, updater := acquireRecord[N](inst, attrs)
	defer rec.refMapped.unref()

	if err := aggregator.RangeTest(num, &rec.instrument.descriptor); err != nil {
		otel.Handle(err)
		return
	}
	updater.(viewstate.AccumulatorUpdater[N]).Update(num)

	// Record was modified.
	atomic.AddInt64(&rec.updateCount, 1)
}

// acquireRecord gets or creates a `*record` corresponding to `kvs`,
// the input labels.  The second argument `labels` is passed in to
// support re-use of the orderedLabels computed by a previous
// measurement in the same batch.   This performs two allocations
// in the common case.
func acquireRecord[N number.Any](inst *Instrument, attrs []attribute.KeyValue) (*record, viewstate.Updater[N]) {
	aset := attribute.NewSet(attrs...)
	if lookup, ok := inst.current.Load(aset); ok {
		// Existing record case.
		rec := lookup.(*record)

		if rec.refMapped.ref() {
			// At this moment it is guaranteed that the
			// record is in the map and will not be removed.
			return rec, rec.accumulator.(viewstate.Updater[N])
		}
		// This group is no longer mapped, try
		// to add a new group below.
	}

	newRec := &record{
		refMapped:  refcountMapped{value: 2},
		instrument: inst,
		distinct:   aset,
		attributes: attrs,
	}

	for {
		if found, loaded := inst.current.LoadOrStore(aset, newRec); loaded {
			oldRec := found.(*record)
			if oldRec.refMapped.ref() {
				return oldRec, oldRec.accumulator.(viewstate.Updater[N])
			}
			runtime.Gosched()
			continue
		}
		break
	}

	return newRec, initRecord[N](inst, newRec, attrs)
}

func initRecord[N number.Any](inst *Instrument, rec *record, attrs []attribute.KeyValue) viewstate.Updater[N] {
	rec.accumulator = inst.compiled.NewAccumulator(attrs, nil)
	return rec.accumulator.(viewstate.Updater[N])
}
