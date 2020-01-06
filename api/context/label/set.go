package label

import (
	"go.opentelemetry.io/otel/api/context/internal"
	"go.opentelemetry.io/otel/api/core"
)

type Set struct {
	set *internal.Set
}

func Empty() Set {
	return Set{internal.EmptySet()}
}

func NewSet(kvs ...core.KeyValue) Set {
	return Set{internal.NewSet(kvs...)}
}

func (s Set) Add1(kv core.KeyValue) Set {
	return Set{s.set.AddOne(kv)}

}

func (s Set) AddN(kvs ...core.KeyValue) Set {
	return Set{s.set.AddMany(kvs...)}
}

func (s Set) Value(k core.Key) (core.Value, bool) {
	return s.set.Value(k)
}

func (s Set) HasValue(k core.Key) bool {
	return s.set.HasValue(k)
}

func (s Set) Len() int {
	return s.set.Len()
}

func (s Set) Ordered() []core.KeyValue {
	return s.set.Ordered()
}

func (s Set) Foreach(f func(kv core.KeyValue) bool) {
	for _, kv := range s.set.Ordered() {
		if !f(kv) {
			return
		}
	}
}

func (s Set) Equals(t Set) bool {
	return s.set.Equals(t.set)
}

func (s Set) Encoded(enc core.LabelEncoder) string {
	return s.set.Encoded(enc)
}
