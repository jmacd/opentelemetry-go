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

package global_test

import (
	"testing"

	"go.opentelemetry.io/otel/api/context/baggage"
	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/context/scope"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
)

type (
	testScopeProvider struct{}
)

var (
	_ scope.Provider = &testScopeProvider{}
)

func (*testScopeProvider) Tracer() trace.Tracer {
	return trace.NoopTracer{}
}

func (*testScopeProvider) Meter() metric.Meter {
	return metric.NoopMeter{}
}

func (*testScopeProvider) Propagators() propagation.Propagators {
	return propagation.New()
}

func (*testScopeProvider) Resources() baggage.Map {
	return baggage.NewEmptyMap()
}

func TestMulitpleGlobalScopeProvider(t *testing.T) {
	p1 := &testScopeProvider{}
	p2 := &testScopeProvider{}
	global.SetScopeProvider(p1)
	global.SetScopeProvider(p2)

	got := global.ScopeProvider()
	want := p2
	if got != want {
		t.Fatalf("Provider: got %p, want %p\n", got, want)
	}
}
