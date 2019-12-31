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

package trace

import (
	"context"

	"go.opentelemetry.io/otel/api/context/scope"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/internal/trace/parent"
)

func (tr *Tracer) Start(ctx context.Context, name string, o ...apitrace.StartOption) (context.Context, apitrace.Span) {
	var opts apitrace.StartConfig

	for _, op := range o {
		op(&opts)
	}

	ctx, parentSpanContext, remoteParent := parent.GetContext(ctx, opts.Parent)

	current := scope.Current(ctx)
	if p := current.Span(); p != nil {
		if sdkSpan, ok := p.(*span); ok {
			sdkSpan.addChild()
		}
	}

	spanName := tr.spanNameWithPrefix(name)
	span := startSpanInternal(tr, spanName, parentSpanContext, remoteParent, opts)
	for _, l := range opts.Links {
		span.addLink(l)
	}
	span.SetAttributes(opts.Attributes...)

	span.tracer = tr

	if span.IsRecording() {
		sps, _ := tr.spanProcessors.Load().(spanProcessorMap)
		for sp := range sps {
			sp.OnStart(span.data)
		}
	}

	ctx, end := startExecutionTracerTask(ctx, spanName)
	span.executionTracerTaskEnd = end
	return scope.ContextWithScope(ctx, current.WithSpan(span)), span
}

func (tr *Tracer) WithSpan(ctx context.Context, name string, body func(ctx context.Context) error) error {
	ctx, span := tr.Start(ctx, name)
	defer span.End()

	if err := body(ctx); err != nil {
		// TODO: set event with boolean attribute for error.
		return err
	}
	return nil
}

func (tr *Tracer) spanNameWithPrefix(name string) string {
	// @@@
	// if tr.name != "" {
	// 	return tr.name + "/" + name
	// }
	return name
}
