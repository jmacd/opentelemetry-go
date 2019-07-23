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

// Tracer return tracer registered with global registry.
// If no tracer is registered then an instance of noop Tracer is returned.
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

// IndirectTracer is used to call the current global tracer after it is initialized.
// Before the global tracer is initialized, this acts as a no-op.
type IndirectTracer struct {
}

var globalIndirect trace.Tracer = &IndirectTracer{}

func (it *IndirectTracer) WithSpan(ctx context.Context, name string, body func(context.Context) error) error {
	return Tracer().WithSpan(ctx, name, body)
}

func (it *IndirectTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

func (it *IndirectTracer) Inject(ctx context.Context, span trace.Span, injector trace.Injector) {
	Tracer().Inject(ctx, span, injector)
}
