package grpc

import (
	"go.opentelemetry.io/api/key"
	"go.opentelemetry.io/api/metric"
	"go.opentelemetry.io/api/metric/aggregation"
	"go.opentelemetry.io/api/unit"
)

// Implements the default metrics export configuration declared here:
// https://github.com/census-instrumentation/opencensus-specs/blob/master/stats/gRPC.md

var (
	// Client metric tags
	clientMethodKey = key.New("grpc_client_method",
		key.WithDescription("Full gRPC method name, including package, service and method, e.g. google.bigtable.v2.Bigtable/CheckAndMutateRow"),
	)

	clientStatusKey = key.New("grpc_client_method",
		key.WithDescription("gRPC server status code received, e.g. OK, CANCELLED, DEADLINE_EXCEEDED"),
	)

	// The default client metrics:
	//
	// grpc.io/client/sent_bytes_per_rpc		distribution	grpc_client_method
	// grpc.io/client/received_bytes_per_rpc	distribution	grpc_client_method
	// grpc.io/client/roundtrip_latency		distribution	grpc_client_method
	// grpc.io/client/completed_rpcs		count		grpc_client_method, grpc_client_status
	// grpc.io/client/started_rpcs			count		grpc_client_method

	sentBytesPerRPC = metric.NewFloat64Measure(
		"grpc.io/client/sent_bytes_per_rpc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Number of bytes sent per RPC"),
		metric.WithAggregations(aggregation.Distribution(clientMethodKey)),
	)

	receivedBytesPerRPC = metric.NewFloat64Measure(
		"grpc.io/client/received_bytes_per_rpc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Number of bytes received per RPC"),
		metric.WithAggregations(aggregation.Distribution(clientMethodKey)),
	)

	roundtripLatency = metric.NewFloat64Measure(
		"grpc.io/client/roundtrip_latency",
		metric.WithUnit(unit.Milliseconds),
		metric.WithDescription("Roundtrip request latency"),
		metric.WithAggregations(aggregation.Distribution(clientMethodKey)),
	)

	// Note: the specification says to use "Count" aggregation by
	// default, which appears to assume that the number of
	// completed_rpcs is always 1.  Shouldn't this use a Sum
	// aggregation?
	//
	// Question: how is this different than the following?
	//     NewFloat64Cumulative(
	//         "grpc.io/client/completed_rpcs",
	//         metric.WithDescription(...),
	//         metric.WithKeys(clientMethodKey, clientStatusKey),
	//     )
	// Answer: these are the effectively the same, i.e., default equivalent default views.
	// The only difference is the method name used to add a measurement.
	completedRPCs = metric.NewFloat64Measure(
		"grpc.io/client/completed_rpcs",
		metric.WithDescription("Count of completed RPCs"),
		metric.WithAggregations(aggregation.Sum(clientMethodKey, clientStatusKey)),
	)

	startedRPCs = metric.NewFloat64Measure(
		"grpc.io/client/started_rpcs",
		metric.WithDescription("Count of started RPCs"),
		metric.WithAggregations(aggregation.Sum(clientMethodKey)),
	)

	// Extra client metrics
	// grpc.io/client/sent_messages_per_rpc		distribution	grpc_client_method
	// grpc.io/client/received_messages_per_rpc	distribution	grpc_client_method
	// grpc.io/client/server_latency		distribution	grpc_client_method
	// grpc.io/client/sent_messages_per_method	count		grpc_client_method
	// grpc.io/client/received_messages_per_method	count		grpc_client_method
	// grpc.io/client/sent_bytes_per_method		sum		grpc_client_method
	// grpc.io/client/received_bytes_per_method	sum		grpc_client_method

	sentMessagesPerRPC = metric.NewFloat64Measure(
		"grpc.io/client/sent_messages_per_rpc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Number of messages sent per RPC, equals 1 for unary RPCs"),
		metric.WithKeys(clientMethodKey),
		metric.WithAggregations(aggregation.None()),
	)

	receivedMessagesPerRPC = metric.NewFloat64Measure(
		"grpc.io/client/received_messages_per_rpc",
		metric.WithUnit(unit.Bytes),
		metric.WithDescription("Number of messages received per RPC, equals 1 for unary RPCs"),
		metric.WithKeys(clientMethodKey),
		metric.WithAggregations(aggregation.None()),
	)

	// ... and so on
)
