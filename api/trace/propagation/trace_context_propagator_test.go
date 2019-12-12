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

package propagation_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/api/context/propagation"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace"
	tpropagation "go.opentelemetry.io/otel/api/trace/propagation"
	mocktrace "go.opentelemetry.io/otel/internal/trace"
)

var (
	traceID = mustTraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID  = mustSpanIDFromHex("00f067aa0ba902b7")
)

func mustTraceIDFromHex(s string) (t core.TraceID) {
	t, _ = core.TraceIDFromHex(s)
	return
}

func mustSpanIDFromHex(s string) (t core.SpanID) {
	t, _ = core.SpanIDFromHex(s)
	return
}

func TestExtractValidTraceContextFromHTTPReq(t *testing.T) {
	props := propagation.New(propagation.WithExtractors(tpropagation.TraceContext{}))
	tests := []struct {
		name   string
		header string
		wantSc core.SpanContext
	}{
		{
			name:   "valid w3cHeader",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00",
			wantSc: core.SpanContext{
				TraceID: traceID,
				SpanID:  spanID,
			},
		},
		{
			name:   "valid w3cHeader and sampled",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantSc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: core.TraceFlagsSampled,
			},
		},
		{
			name:   "future version",
			header: "02-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantSc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: core.TraceFlagsSampled,
			},
		},
		{
			name:   "future options with sampled bit set",
			header: "02-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-09",
			wantSc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: core.TraceFlagsSampled,
			},
		},
		{
			name:   "future options with sampled bit cleared",
			header: "02-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-08",
			wantSc: core.SpanContext{
				TraceID: traceID,
				SpanID:  spanID,
			},
		},
		{
			name:   "future additional data",
			header: "02-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-09-XYZxsf09",
			wantSc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: core.TraceFlagsSampled,
			},
		},
		{
			name:   "valid b3Header ending in dash",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-",
			wantSc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: core.TraceFlagsSampled,
			},
		},
		{
			name:   "future valid b3Header ending in dash",
			header: "01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-09-",
			wantSc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: core.TraceFlagsSampled,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			req.Header.Set("traceparent", tt.header)

			ctx := context.Background()
			ctx = propagation.ExtractHTTP(ctx, props, req.Header)
			gotSc := tpropagation.UpstreamContext(ctx)
			if diff := cmp.Diff(gotSc, tt.wantSc); diff != "" {
				t.Errorf("Extract Tracecontext: %s: -got +want %s", tt.name, diff)
			}
		})
	}
}

func TestExtractInvalidTraceContextFromHTTPReq(t *testing.T) {
	wantSc := core.EmptySpanContext()
	props := propagation.New(propagation.WithExtractors(tpropagation.TraceContext{}))
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "wrong version length",
			header: "0000-00000000000000000000000000000000-0000000000000000-01",
		},
		{
			name:   "wrong trace ID length",
			header: "00-ab00000000000000000000000000000000-cd00000000000000-01",
		},
		{
			name:   "wrong span ID length",
			header: "00-ab000000000000000000000000000000-cd0000000000000000-01",
		},
		{
			name:   "wrong trace flag length",
			header: "00-ab000000000000000000000000000000-cd00000000000000-0100",
		},
		{
			name:   "bogus version",
			header: "qw-00000000000000000000000000000000-0000000000000000-01",
		},
		{
			name:   "bogus trace ID",
			header: "00-qw000000000000000000000000000000-cd00000000000000-01",
		},
		{
			name:   "bogus span ID",
			header: "00-ab000000000000000000000000000000-qw00000000000000-01",
		},
		{
			name:   "bogus trace flag",
			header: "00-ab000000000000000000000000000000-cd00000000000000-qw",
		},
		{
			name:   "upper case version",
			header: "A0-00000000000000000000000000000000-0000000000000000-01",
		},
		{
			name:   "upper case trace ID",
			header: "00-AB000000000000000000000000000000-cd00000000000000-01",
		},
		{
			name:   "upper case span ID",
			header: "00-ab000000000000000000000000000000-CD00000000000000-01",
		},
		{
			name:   "upper case trace flag",
			header: "00-ab000000000000000000000000000000-cd00000000000000-A1",
		},
		{
			name:   "zero trace ID and span ID",
			header: "00-00000000000000000000000000000000-0000000000000000-01",
		},
		{
			name:   "trace-flag unused bits set",
			header: "00-ab000000000000000000000000000000-cd00000000000000-09",
		},
		{
			name:   "missing options",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7",
		},
		{
			name:   "empty options",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			req.Header.Set("traceparent", tt.header)

			ctx := context.Background()
			ctx = propagation.ExtractHTTP(ctx, props, req.Header)
			gotSc := tpropagation.UpstreamContext(ctx)
			if diff := cmp.Diff(gotSc, wantSc); diff != "" {
				t.Errorf("Extract Tracecontext: %s: -got +want %s", tt.name, diff)
			}
		})
	}
}

func TestInjectTraceContextToHTTPReq(t *testing.T) {
	var id uint64
	mockTracer := &mocktrace.MockTracer{
		Sampled:     false,
		StartSpanID: &id,
	}
	props := propagation.New(propagation.WithInjectors(tpropagation.TraceContext{}))
	tests := []struct {
		name       string
		sc         core.SpanContext
		wantHeader string
	}{
		{
			name: "valid spancontext, sampled",
			sc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: core.TraceFlagsSampled,
			},
			wantHeader: "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000001-01",
		},
		{
			name: "valid spancontext, not sampled",
			sc: core.SpanContext{
				TraceID: traceID,
				SpanID:  spanID,
			},
			wantHeader: "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000002-00",
		},
		{
			name: "valid spancontext, with unsupported bit set in traceflags",
			sc: core.SpanContext{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0xff,
			},
			wantHeader: "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000003-01",
		},
		{
			name:       "invalid spancontext",
			sc:         core.EmptySpanContext(),
			wantHeader: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			ctx := context.Background()
			if tt.sc.IsValid() {
				ctx, _ = mockTracer.Start(ctx, "inject", trace.WithParent(tpropagation.WithUpstreamContext(ctx, tt.sc)))
			}
			propagation.InjectHTTP(ctx, props, req.Header)

			gotHeader := req.Header.Get("traceparent")
			if diff := cmp.Diff(gotHeader, tt.wantHeader); diff != "" {
				t.Errorf("Extract Tracecontext: %s: -got +want %s", tt.name, diff)
			}
		})
	}
}

func TestTraceContextPropagator_GetAllKeys(t *testing.T) {
	var propagator tpropagation.TraceContext
	want := []string{"Traceparent"}
	got := propagator.GetAllKeys()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("GetAllKeys: -got +want %s", diff)
	}
}
