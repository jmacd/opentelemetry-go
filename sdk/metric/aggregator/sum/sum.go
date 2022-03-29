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

package sum // import "go.opentelemetry.io/otel/sdk/metric/aggregator/sum"

import (
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/metric/number"
)

type (
	Config struct{}

	Methods[N number.Any, Storage State[N]] struct{}

	State[N number.Any] struct {
		value N
	}
)

var (
	_ aggregator.Methods[int64, State[int64], Config]     = Methods[int64, State[int64]]{}
	_ aggregator.Methods[float64, State[float64], Config] = Methods[float64, State[float64]]{}

	_ aggregation.Sum = &State[int64]{}
	_ aggregation.Sum = &State[float64]{}
)

func (s *State[N]) Sum() (number.Number, error) {
	return number.ToNumber(s.value), nil
}

func (s *State[N]) Kind() aggregation.Kind {
	return aggregation.SumKind
}

func (Methods[N, Storage]) Init(state *State[N], _ Config) {
	// Note: storage is zero to start
}

func (Methods[N, Storage]) SynchronizedMove(resetSrc, dest *State[N]) {
	if dest == nil {
		number.SetAtomic(&resetSrc.value, 0)
		return
	}
	dest.value = number.SwapAtomic(&resetSrc.value, 0)
}

func (Methods[N, Storage]) Update(state *State[N], value N) {
	number.AddAtomic(&state.value, value)
}

func (Methods[N, Storage]) Merge(to, from *State[N]) {
	to.value += from.value
}

func (Methods[N, Storage]) Aggregation(state *State[N]) aggregation.Aggregation {
	return state
}

func (Methods[N, Storage]) Storage(aggr aggregation.Aggregation) *State[N] {
	return aggr.(*State[N])
}

func (Methods[N, Storage]) Subtract(valueToModify, operand *State[N]) error {
	valueToModify.value -= operand.value
	return nil
}
