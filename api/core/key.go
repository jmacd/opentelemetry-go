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
	"io"
	"unsafe"

	"go.opentelemetry.io/api/registry"
)

type Key struct {
	Variable registry.Variable
}

type KeyValue struct {
	Key   Key
	Value Value
}

type ValueType int

type Encoder func(io.Writer) error

type Value struct {
	Type    ValueType
	Bool    bool
	Int64   int64
	Uint64  uint64
	Float64 float64
	String  string
	Bytes   []byte
	Struct  interface{}
	Encoder Encoder
}

const (
	INVALID ValueType = iota
	BOOL
	INT32
	INT64
	UINT32
	UINT64
	FLOAT32
	FLOAT64
	STRING
	BYTES
	STRUCT  // Struct or Bytes, whichever is non-nil.
	ENCODER // Encoder or Bytes, whichever is non-nil.
)

func (k Key) Bool(v bool) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type: BOOL,
			Bool: v,
		},
	}
}

func (k Key) Int64(v int64) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:  INT64,
			Int64: v,
		},
	}
}

func (k Key) Uint64(v uint64) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:   UINT64,
			Uint64: v,
		},
	}
}

func (k Key) Float64(v float64) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:    FLOAT64,
			Float64: v,
		},
	}
}

func (k Key) Int32(v int32) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:  INT32,
			Int64: int64(v),
		},
	}
}

func (k Key) Uint32(v uint32) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:   UINT32,
			Uint64: uint64(v),
		},
	}
}

func (k Key) Float32(v float32) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:    FLOAT32,
			Float64: float64(v),
		},
	}
}

func (k Key) String(v string) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:   STRING,
			String: v,
		},
	}
}

func (k Key) Bytes(v []byte) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:  BYTES,
			Bytes: v,
		},
	}
}

func (k Key) Int(v int) KeyValue {
	if unsafe.Sizeof(v) == 4 {
		return k.Int32(int32(v))
	}
	return k.Int64(int64(v))
}

func (k Key) Uint(v uint) KeyValue {
	if unsafe.Sizeof(v) == 4 {
		return k.Uint32(uint32(v))
	}
	return k.Uint64(uint64(v))
}

func (k Key) Struct(v interface{}) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:   STRUCT,
			Struct: v,
		},
	}
}

func (k Key) Encode(v func(io.Writer) error) KeyValue {
	return KeyValue{
		Key: k,
		Value: Value{
			Type:    ENCODER,
			Encoder: v,
		},
	}
}

func (k Key) Defined() bool {
	return k.Variable.Defined()
}

func (v Value) Evaluate() Value {
	switch v.Type {
	case STRUCT:
		if v.Struct != nil {
			return Value{
				Type:  STRUCT,
				Bytes: encodeStruct(v.Struct),
			}
		}
	case ENCODER:
		if v.Encoder != nil {
			return Value{
				Type:  STRUCT,
				Bytes: applyEncoder(v.Encoder),
			}
		}
	}
	return v
}

func (v Value) Emit() string {
	switch v.Type {
	case BOOL:
		return fmt.Sprint(v.Bool)
	case INT32, INT64:
		return fmt.Sprint(v.Int64)
	case UINT32, UINT64:
		return fmt.Sprint(v.Uint64)
	case FLOAT32, FLOAT64:
		return fmt.Sprint(v.Float64)
	case STRING:
		return v.String
	case BYTES, STRUCT, ENCODER:
		// Note: In case of a fully synchronous SDK, this call
		// could be the first to evaluate a struct/encoder value.
		return string(v.Evaluate().Bytes)
	}
	return "invalid"
}
