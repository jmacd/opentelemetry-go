package core

type LabelEncoder interface {
	// Encode is called (concurrently) in instrumentation context.
	// It should return a unique representation of the labels
	// suitable for the SDK to use as a map key, an aggregator
	// grouping key, and/or the export encoding.
	Encode([]KeyValue) string
}
