package trace

type SpanReader interface {
	register()

	WriteOnEndSpanProcessor
	Component
}
