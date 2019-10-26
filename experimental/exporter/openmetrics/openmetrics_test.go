package openmetrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/api/key"
	sdk "go.opentelemetry.io/sdk/metric"
)

func TestBasicExporter(t *testing.T) {
	ctx := context.Background()
	sdk := sdk.New(NewOpenMetricsExporter())

	cnt := sdk.NewInt64Counter("counter.x")

	cnt.Add(ctx, 100, sdk.Labels(ctx, key.String("l1", "v1")))

	sdk.Collect(ctx)
}
