package openmetrics

import (
	"bytes"
	"context"
	"testing"

	"go.opentelemetry.io/api/key"
	sdk "go.opentelemetry.io/sdk/metric"
)

func TestBasicExporter(t *testing.T) {
	ctx := context.Background()
	ome := NewExporter()
	sdk := sdk.New(ome)

	cnt := sdk.NewInt64Counter("counter.x")

	cnt.Add(ctx, 100, sdk.Labels(key.String("l1", "v1")))

	sdk.Collect(ctx)

	var buf bytes.Buffer

	if _, err := ome.Write(&buf); err != nil {
		t.Fatal("Write failed: ", err)
	}

}
