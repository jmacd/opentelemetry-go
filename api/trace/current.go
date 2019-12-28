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
)

type currentSpanKeyType struct{}

var currentSpanKey = &currentSpanKeyType{}

func ContextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, currentSpanKey, span)
}

func SpanFromContext(ctx context.Context) Span {
	if span, has := ctx.Value(currentSpanKey).(Span); has {
		return span
	}
	return NoopSpan{}
}

// @@@
// We seem to want a current
// and Name(), WithLabel()... ...
// Tracer(ctx).Span()
// and Meter(ctx).NewInt64Counter()
// and Namespaces
// and Labels(...)
// and LabelEncoder()
// and ...
