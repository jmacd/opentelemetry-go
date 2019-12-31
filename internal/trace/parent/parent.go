package parent

import (
	"context"

	"go.opentelemetry.io/otel/api/context/scope"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace/propagation"
)

func getEffective(ctx context.Context) (core.SpanContext, bool) {
	scc := scope.Current(ctx)
	scp := scc.Span()
	if sctx := scp.SpanContext(); sctx.IsValid() {
		return sctx, false
	}
	return propagation.RemoteContext(ctx), true
}

func GetContext(ctx, parent context.Context) (context.Context, core.SpanContext, bool) {
	if parent != nil {
		pctx, remote := getEffective(parent)
		return parent, pctx, remote
	}

	sctx, remote := getEffective(ctx)
	return ctx, sctx, remote
}
