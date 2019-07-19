package grpc

import (
	"go.opentelemetry.io/api/key"
	"go.opentelemetry.io/api/metric/aggregation"
	"go.opentelemetry.io/api/unit"
)

// Discussion
// https://github.com/census-instrumentation/opencensus-specs/blob/master/stats/gRPC.md

var (
	// Client metrics
	clientMethodKey = key.New("grpc_client_method",
		key.WithDescription("Full gRPC method name, including package, service and method, e.g. google.bigtable.v2.Bigtable/CheckAndMutateRow"),
	)

	clientStatusKey = key.New("grpc_client_method",
		key.WithDescription("gRPC server status code received, e.g. OK, CANCELLED, DEADLINE_EXCEEDED"),
	)

	sentMessagesPerRPC = metric.NewFloat64Measure(
		"grpc.io/client/sent_messages_per_rpc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Number of messages sent per RPC, equals 1 for unary RPCs"),
		metric.WithAggregation(aggregation.Distribution(clientMethodKey)),
	)

	receivedMessagesPerRPC = metric.NewFloat64Measure(
		"grpc.io/client/received_messages_per_rpc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Number of messages received per RPC, equals 1 for unary RPCs"),
		metric.WithAggregation(aggregation.Distribution(clientMethodKey)),
	)

	receivedMessagesPerRPC = metric.NewFloat64Measure(
		"grpc.io/client/received_messages_per_rpc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Number of messages received per RPC, equals 1 for unary RPCs"),
		metric.WithAggregation(aggregation.Distribution(clientMethodKey)),
	)
)
