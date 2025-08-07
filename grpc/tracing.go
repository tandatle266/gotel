package grpc

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

func InstrumentGRPCServerOptions(opts ...grpc.ServerOption) []grpc.ServerOption {
    newOpts := make([]grpc.ServerOption, 0, len(opts)+1)
    newOpts = append(newOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
    newOpts = append(newOpts, opts...)
    return newOpts
}


func InstrumentGRPCDialOptions(opts ...grpc.DialOption) []grpc.DialOption {
    newOpts := make([]grpc.DialOption, 0, len(opts)+1)
    newOpts = append(newOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
    newOpts = append(newOpts, opts...)
    return newOpts
}

func StatsHandlerOption() grpc.ServerOption {
    return grpc.StatsHandler(otelgrpc.NewServerHandler())
}
