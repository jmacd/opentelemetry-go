// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute // import "go.opentelemetry.io/otel/attribute"

import (
	"fmt"
)

// KeyValue holds a key and value pair.
type KeyValue struct {
	Key   Key
	Value Value
}

// Valid returns if kv is a valid OpenTelemetry attribute.
func (kv KeyValue) Valid() bool {
	return kv.Key.Defined() && kv.Value.Type() != INVALID
}

// Bool creates a KeyValue with a BOOL Value type.
func Bool(k string, v bool) KeyValue {
	return Key(k).Bool(v)
}

// BoolSlice creates a KeyValue with a BOOLSLICE Value type.
func BoolSlice(k string, v []bool) KeyValue {
	return Key(k).BoolSlice(v)
}

// Int creates a KeyValue with an INT64 Value type.
func Int(k string, v int) KeyValue {
	return Key(k).Int(v)
}

// IntSlice creates a KeyValue with an INT64SLICE Value type.
func IntSlice(k string, v []int) KeyValue {
	return Key(k).IntSlice(v)
}

// Int64 creates a KeyValue with an INT64 Value type.
func Int64(k string, v int64) KeyValue {
	return Key(k).Int64(v)
}

// Int64Slice creates a KeyValue with an INT64SLICE Value type.
func Int64Slice(k string, v []int64) KeyValue {
	return Key(k).Int64Slice(v)
}

// Float64 creates a KeyValue with a FLOAT64 Value type.
func Float64(k string, v float64) KeyValue {
	return Key(k).Float64(v)
}

// Float64Slice creates a KeyValue with a FLOAT64SLICE Value type.
func Float64Slice(k string, v []float64) KeyValue {
	return Key(k).Float64Slice(v)
}

// String creates a KeyValue with a STRING Value type.
func String(k, v string) KeyValue {
	return Key(k).String(v)
}

// StringSlice creates a KeyValue with a STRINGSLICE Value type.
func StringSlice(k string, v []string) KeyValue {
	return Key(k).StringSlice(v)
}

// Bytes creates a KeyValue with a BYTES Value type.
func Bytes(k string, v []byte) KeyValue {
	return Key(k).Bytes(v)
}

// Stringer creates a new key-value pair with a passed name and a string
// value generated by the passed Stringer interface.
func Stringer(k string, v fmt.Stringer) KeyValue {
	return Key(k).String(v.String())
}
