# gotel

A lightweight Go toolkit for integrating OpenTelemetry tracing into your gRPC clients, servers, and Redis clients.

---

## Features

- **gRPC client wrappers** with built-in OpenTelemetry tracing interceptors  
- **gRPC server setup** with interceptors and stats handler for tracing and metrics  
- **Redis client wrapper** with OpenTelemetry tracing support  
- **Minimal boilerplate**, reusable, and easy to integrate  
- **Seamless context propagation** across services  

---

## Installation

```bash
go get github.com/tandatle266/gotracer
```

---

## Usage

### 1. Create a gRPC client with tracing wrapper

Create gRPC clients that automatically include OpenTelemetry tracing interceptors:

```go
serviceClient, err := grpc.NewClientWithTracing("localhost:port", "server-service")
if err != nil {
    logger.Error("Failed to initialize gRPC setting client: " + err.Error())
    return
}
defer serviceClient.Close()

// Example RPC call
resp, err := serviceClient.client.SomeRPC(ctx, &proto.SomeRequest{})
if err != nil {
    logger.Error("RPC call failed: " + err.Error())
}
```

### 2. Create a gRPC server with tracing interceptors and stats handler

Initialize a gRPC server configured with OpenTelemetry interceptors for tracing unary and streaming RPCs, plus a stats handler for metrics.

```go
handler := otelgrpc.NewServerHandler()

grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        otelgrpc.UnaryServerInterceptor(),
        pkgGrpc.RecoveryInterceptor,
    ),
    grpc.ChainStreamInterceptor(
        otelgrpc.StreamServerInterceptor(),
    ),
    grpc.StatsHandler(handler),
)

// Register your services and start serving
proto.RegisterClientServiceServer(grpcServer, yourServiceImplementation)

lis, err := net.Listen("tcp", ":port")
if err != nil {
    logger.Error("failed to listen: " + err.Error())
}
go func() {
    if err := grpcServer.Serve(lis); err != nil {
        logger.Error("gRPC server failed: " + err.Error())
    }
}()
```

### 3. Use Redis client wrapper with tracing

Create a Redis client wrapped with OpenTelemetry tracing support.

```go
redisOptions := &redis.Options{
    Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
    Password: cfg.RedisPassword,
    DB:       cfg.RedisDB,
}

redisClient := oteltracingredis.NewClient(redisOptions, "server-service")
defer redisClient.Close()

// Use redisClient.Client() for normal Redis commands
```

---

## Summary

- Use gRPC client wrappers to automatically include tracing interceptors when dialing services.  
- Configure gRPC servers with tracing interceptors and stats handler for comprehensive tracing and metrics.  
- Wrap Redis clients to capture Redis commands within traces.  
- Ensure proper context propagation for full distributed tracing across services.  

---

## Contributing

Contributions and suggestions are welcome! Please open issues or submit pull requests.

---

## License

MIT License
