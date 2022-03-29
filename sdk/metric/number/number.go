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

package number

import (
	"math"
	"sync/atomic"
	"unsafe"
)

//go:generate stringer -type=Kind

// Kind describes the data type of the Number.
type Kind int8

const (
	// Int64Kind means that the Number stores int64.
	Int64Kind Kind = iota

	// Float64Kind means that the Number stores float64.
	Float64Kind
)

// Number is a generic 64bit numeric value.
type Number uint64

// Any is any of the supported generic Number types.
type Any interface {
	int64 | float64
}

func ToNumber[N Any](value N) Number {
	switch n := any(value).(type) {
	case int64:
		return Number(n)
	case float64:
		return Number(math.Float64bits(n))
	}
	return Number(0)
}

func SetAtomic[N Any](ptr *N, value N) {
	switch v := any(value).(type) {
	case int64:
		p := any(ptr).(*int64)
		atomic.StoreInt64(p, v)
	case float64:
		atomic.StoreUint64((*uint64)(unsafe.Pointer(ptr)), math.Float64bits(v))
	}
}
func AddAtomic[N Any](ptr *N, value N) {
	panic("here")
}

func SwapAtomic[N Any](ptr *N, value N) N {
	switch v := any(value).(type) {
	case int64:
		p := any(ptr).(*int64)
		return N(atomic.SwapInt64(p, v))
	case float64:
		return N(math.Float64frombits(atomic.SwapUint64((*uint64)(unsafe.Pointer(ptr)), math.Float64bits(v))))
	}
	return N(0)
}
func IsNaN[N Any](value N) bool {
	switch n := any(value).(type) {
	case float64:
		return math.IsNaN(n)
	}
	return false
}
func IsInf[N Any](value N) bool {
	switch n := any(value).(type) {
	case float64:
		return math.IsInf(n, 0)
	}
	return false
}
