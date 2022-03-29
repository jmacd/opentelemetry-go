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

package lastvalue // import "go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"

import (
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/metric/number"
)

var ErrNoSubtract = fmt.Errorf("lastvalue subtract not implemented")

type (
	Config struct{}

	Methods[N number.Any, Storage State[N]] struct{}

	State[N number.Any] struct {
		lock      sync.Mutex
		value     N
		timestamp time.Time
	}
)

var (
	_ aggregator.Methods[int64, State[int64], Config]     = Methods[int64, State[int64]]{}
	_ aggregator.Methods[float64, State[float64], Config] = Methods[float64, State[float64]]{}

	_ aggregation.LastValue = &State[int64]{}
	_ aggregation.LastValue = &State[float64]{}
)

// LastValue returns the last-recorded lastValue value and the
// corresponding timestamp.  The error value aggregation.ErrNoData
// will be returned if (due to a race condition) the checkpoint was
// computed before the first value was set.
func (lv *State[N]) LastValue() (number.Number, time.Time, error) {
	lv.lock.Lock()
	defer lv.lock.Unlock()
	if lv.timestamp.IsZero() {
		return 0, time.Time{}, aggregation.ErrNoData
	}
	return number.ToNumber(lv.value), lv.timestamp, nil
}

func (lv *State[N]) Kind() aggregation.Kind {
	return aggregation.LastValueKind
}

func (Methods[N, Storage]) Init(state *State[N], _ Config) {
	// Note: storage is zero to start
}

func (Methods[N, Storage]) SynchronizedMove(resetSrc, dest *State[N]) {
	resetSrc.lock.Lock()
	defer resetSrc.lock.Unlock()

	if dest != nil {
		dest.value = resetSrc.value
		dest.timestamp = resetSrc.timestamp
	}
	resetSrc.value = 0
	resetSrc.timestamp = time.Time{}
}

func (Methods[N, Storage]) Update(state *State[N], number N) {
	now := time.Now()

	state.lock.Lock()
	defer state.lock.Unlock()

	state.value = number
	state.timestamp = now
}

func (Methods[N, Storage]) Merge(to, from *State[N]) {
	if to.timestamp.After(from.timestamp) {
		return
	}

	to.value = from.value
	to.timestamp = from.timestamp
}

func (Methods[N, Storage]) Aggregation(state *State[N]) aggregation.Aggregation {
	return state
}

func (Methods[N, Storage]) Storage(aggr aggregation.Aggregation) *State[N] {
	return aggr.(*State[N])
}

func (Methods[N, Storage]) Subtract(valueToModify, operand *State[N]) error {
	return ErrNoSubtract
}
