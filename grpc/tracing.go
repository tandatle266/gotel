package oteltracinggrpc

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientFactory[T any] func(cc grpc.ClientConnInterface) T

// NewClientWithTracing create a gRPC client with tracing
func NewClientWithTracing[T any](addr string, factory ClientFactory[T]) (client T, conn *grpc.ClientConn, err error) {
    dialOptions := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
    }

    conn, err = grpc.Dial(addr, dialOptions...)
    if err != nil {
        return client, nil, err
    }

    client = factory(conn)
    return client, conn, nil
}