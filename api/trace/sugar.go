package trace

import (
	"context"

	"go.opentelemetry.io/otel/api/internal"
)

type provider interface {
	Tracer() Tracer
}

func getTracer(ctx context.Context) Tracer {
	if p, ok := internal.ScopeImpl(ctx).(provider); ok {
		return p.Tracer()
	}

	if g, ok := internal.GlobalScope.Load().(provider); ok {
		return g.Tracer()
	}

	return NoopTracer{}
}

func Start(ctx context.Context, spanName string, opts ...StartOption) (context.Context, Span) {
	tracer := getTracer(ctx)
	return tracer.Start(ctx, spanName, opts...)
}

func WithSpan(ctx context.Context, spanName string, fn func(ctx context.Context) error) error {
	tracer := getTracer(ctx)
	return tracer.WithSpan(ctx, spanName, fn)
}
