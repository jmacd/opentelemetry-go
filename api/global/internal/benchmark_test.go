package internal_test

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/api/context/scope"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/global/internal"
	"go.opentelemetry.io/otel/api/key"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/counter"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/ddsketch"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/gauge"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
)

// benchFixture is copied from sdk/metric/benchmark_test.go.
// TODO refactor to share this code.
type benchFixture struct {
	sdk *sdk.SDK
	B   *testing.B
}

func newFixture(b *testing.B) *benchFixture {
	b.ReportAllocs()
	bf := &benchFixture{
		B: b,
	}
	bf.sdk = sdk.New(bf, sdk.NewDefaultLabelEncoder())
	return bf
}

func (*benchFixture) AggregatorFor(descriptor *export.Descriptor) export.Aggregator {
	switch descriptor.MetricKind() {
	case export.CounterKind:
		return counter.New()
	case export.GaugeKind:
		return gauge.New()
	case export.MeasureKind:
		if strings.HasSuffix(descriptor.Name(), "minmaxsumcount") {
			return minmaxsumcount.New(descriptor)
		} else if strings.HasSuffix(descriptor.Name(), "ddsketch") {
			return ddsketch.New(ddsketch.NewDefaultConfig(), descriptor)
		} else if strings.HasSuffix(descriptor.Name(), "array") {
			return ddsketch.New(ddsketch.NewDefaultConfig(), descriptor)
		}
	}
	return nil
}

func (*benchFixture) Process(context.Context, export.Record) error {
	return nil
}

func (*benchFixture) CheckpointSet() export.CheckpointSet {
	return nil
}

func (*benchFixture) FinishedCollection() {
}

func BenchmarkGlobalInt64CounterAddNoSDK(b *testing.B) {
	internal.ResetForTest()
	ctx := context.Background()
	sdk := global.Scope("test").Meter()
	labs := core.NewLabels(key.String("A", "B"))
	cnt := sdk.NewInt64Counter("int64.counter")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt.Add(ctx, 1, labs)
	}
}

func BenchmarkGlobalInt64CounterAddWithSDK(b *testing.B) {
	// Comapare with BenchmarkInt64CounterAdd() in ../../sdk/meter/benchmark_test.go
	ctx := context.Background()
	fix := newFixture(b)

	sdk := global.Scope("test").Meter()

	global.SetScopeProvider(scope.NewProvider(nil, fix.sdk, nil))

	labs := core.NewLabels(key.String("A", "B"))
	cnt := sdk.NewInt64Counter("int64.counter")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt.Add(ctx, 1, labs)
	}
}
