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

package global

import (
	"context"

	"go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/api/trace/internal"
)

// Tracer returns the global Tracer instance.  Before opentelemetry.Init() is
// called, this returns an "indirect" No-op implementation, that will be updated
// once the SDK is initialized.
func Tracer() trace.Tracer {
	if t := internal.Global.Load(); t != nil {
		return t.(trace.Tracer)
	}
	return globalIndirect
}

// SetTracer sets provided tracer as a global tracer.
func SetTracer(t trace.Tracer) {
	internal.Global.Store(t)
}

// IndirectTracer implements Tracer, allows callers of Tracer() before Init()
// to forward to the installed SDK.
type IndirectTracer struct{}

var globalIndirect trace.Tracer = &IndirectTracer{}

func (*IndirectTracer) WithSpan(
	ctx context.Context,
	name string,
	body func(context.Context) error,
) error {
	return Tracer().WithSpan(ctx, name, body)
}

func (*IndirectTracer) Start(
	ctx context.Context,
	name string,
	opts ...trace.SpanOption,
) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

func (*IndirectTracer) Inject(
	ctx context.Context,
	span trace.Span,
	injector trace.Injector,
) {
	Tracer().Inject(ctx, span, injector)
}
