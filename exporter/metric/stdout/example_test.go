package stdout_test

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/api/context/scope"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/exporter/metric/stdout"
)

func ExampleNewExportPipeline() {
	// Create a meter
	pusher, err := stdout.NewExportPipeline(stdout.Config{
		PrettyPrint:    true,
		DoNotPrintTime: true,
	})
	if err != nil {
		log.Fatal("Could not initialize stdout exporter:", err)
	}
	defer pusher.Stop()

	ctx := context.Background()

	key := key.New("key")
	scx := scope.Empty().WithMeter(pusher.Meter()).Named("example")

	// Create and update a single counter:
	counter := scx.Meter().NewInt64Counter("a.counter", metric.WithKeys(key))
	labels := core.NewLabels(key.String("value"))

	counter.Add(ctx, 100, labels)

	// Output:
	// {
	// 	"updates": [
	// 		{
	// 			"name": "example/a.counter{key=value}",
	// 			"sum": 100
	// 		}
	// 	]
	// }
}
