package openmetrics

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/api/key"
	"go.opentelemetry.io/api/metric"
	sdk "go.opentelemetry.io/sdk/metric"
)

func TestCounter1(t *testing.T) {
	ctx := context.Background()
	ome := NewExporter()
	sdk := sdk.New(ome)

	cnt := sdk.NewInt64Counter("counter.x", metric.WithKeys("l1"))

	cnt.Add(ctx, 100, sdk.Labels(key.String("l1", "v1")))

	sdk.Collect(ctx)

	var buf bytes.Buffer

	if _, err := ome.Write(&buf); err != nil {
		t.Fatal("Write failed: ", err)
	}

	require.Equal(t, "counter.x{l1=\"v1\"} 100\n", buf.String(), "Counter export match")
}

func TestGauge1(t *testing.T) {
	ctx := context.Background()
	ome := NewExporter()
	sdk := sdk.New(ome)

	cnt := sdk.NewFloat64Gauge("gauge.x", metric.WithKeys("l1"))

	cnt.Set(ctx, 111.111111112, sdk.Labels(key.String("l1", "v1")))

	sdk.Collect(ctx)

	var buf bytes.Buffer

	if _, err := ome.Write(&buf); err != nil {
		t.Fatal("Write failed: ", err)
	}

	require.Equal(t, "gauge.x{l1=\"v1\"} 111.111111112\n", buf.String(), "Gauge export match")
}

func TestMeasure1(t *testing.T) {
	const (
		count  = 100
		values = 3
	)
	var (
		base = math.Sqrt(17)
	)
	ctx := context.Background()
	ome := NewExporter()
	sdk := sdk.New(ome)

	cnt := sdk.NewFloat64Measure("measure.x", metric.WithKeys("v"))

	var labels []metric.LabelSet

	for v := 0; v < values; v++ {
		labels = append(labels, sdk.Labels(key.Int("v", v)))
	}

	sums := make([]float64, values)
	maxs := make([]float64, values)

	for i := 0; i < count; i++ {
		for v := 0; v < values; v++ {
			x := math.Pow(base, float64(i))
			sums[v] += x
			if maxs[v] < x {
				maxs[v] = x
			}
			cnt.Record(ctx, x, labels[v])
		}
	}

	sdk.Collect(ctx)

	var buf bytes.Buffer

	if _, err := ome.Write(&buf); err != nil {
		t.Fatal("Write failed: ", err)
	}

	actual := map[string]bool{}
	expect := map[string]bool{}

	for _, line := range strings.Split(buf.String(), "\n") {
		if line == "" {
			continue
		}
		actual[line] = true
	}

	for v := 0; v < values; v++ {
		expect[fmt.Sprintf("measure.x{v=\"%d\",quantile=\"1\"} %g", v, maxs[v])] = true
		expect[fmt.Sprintf("measure.x_sum{v=\"%d\"} %g", v, sums[v])] = true
		expect[fmt.Sprintf("measure.x_count{v=\"%d\"} %d", v, count)] = true
	}

	require.EqualValues(t, expect, actual, "Gauge export match")

}
