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

package aggregator // import "go.opentelemetry.io/otel/sdk/metric/aggregator"

import (
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/number"
	"go.opentelemetry.io/otel/sdk/metric/sdkapi"
)

// RangeTest is a common routine for testing for valid input values.
// This rejects NaN and Inf values.  This rejects negative values when the
// metric instrument does not support negative values, including
// monotonic counter metrics and absolute Histogram metrics.
func RangeTest[N number.Any](num N, desc *sdkapi.Descriptor) error {

	if number.IsInf(num) {
		return aggregation.ErrInfInput
	}

	if number.IsNaN(num) {
		return aggregation.ErrNaNInput
	}

	// Check for negative values
	switch desc.InstrumentKind() {
	case sdkapi.CounterInstrumentKind,
		sdkapi.CounterObserverInstrumentKind,
		sdkapi.HistogramInstrumentKind:
		if num < 0 {
			return aggregation.ErrNegativeInput
		}
	}
	return nil
}

// Methods implements a specific aggregation behavior.  Methods
// are parameterized by the type of the number (int64, flot64),
// the Storage (generally an `Storage` struct in the same package),
// and the Config (generally a `Config` struct in the same package).
type Methods[N number.Any, Storage, Config any] interface {
	// Init initializes the storage with its configuration.
	Init(ptr *Storage, cfg Config)

	// Update modifies the aggregator concurrently with respect to
	// SynchronizedMove() for single new measurement.
	Update(ptr *Storage, number N)

	// SynchronizedMove concurrently copies and resets the
	// `inputIsReset` aggregator.  If output is non-nil it
	// receives the copy.  If output is nil, it is ignored, which
	// makes `SynchronizedMove(ptr, nil)` equivalent to a Reset
	// operation.
	SynchronizedMove(inputIsReset, output *Storage)

	// Merge adds the contents of `input` to `output`.
	Merge(output, input *Storage)

	// Subtract removes the contents of `operand` from `valueToModify`
	Subtract(valueToModify, operand *Storage) error

	// Aggregation returns an exporter-ready value.
	Aggregation(ptr *Storage) aggregation.Aggregation
}
