# gotel

A lightweight Go toolkit for integrating OpenTelemetry tracing into your gRPC clients, servers, Gin, and Redis clients.

---

## Features

* **Easy gRPC client and server tracing:**
  Add distributed tracing to any gRPC client or server with just one line.
* **Redis tracing support:**
  Instrument any go-redis client for end-to-end trace visibility.
* **Gin HTTP tracing:**
  Simple Gin middleware for HTTP API tracing.
* **Minimal boilerplate:**
  No wrappers or interface rewrites required – just call an instrument function.
* **Seamless context propagation** across microservices.
* **Plug-and-play:**
  Designed for drop-in integration into existing Go codebases.

---

## Installation

```bash
go get github.com/tandatle266/gotel
```

---

## Usage

### 1. Instrument a gRPC Client for Tracing

Enable tracing for a gRPC client connection with a single line:

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "github.com/tandatle266/gotel/grpc/tracing"
)

opts := []grpc.DialOption{
    grpc.WithTransportCredentials(insecure.NewCredentials()), // Or your preferred credentials
}
opts = tracing.InstrumentGRPCDialOptions(opts...)

conn, err := grpc.Dial("localhost:50051", opts...)
if err != nil {
    // handle error
}
defer conn.Close()

client := pb.NewYourServiceClient(conn)
// Use client as normal
```

---

### 2. Instrument a gRPC Server for Tracing

Add tracing to a gRPC server by including a stats handler (and any interceptors you want):

```go
import (
    "google.golang.org/grpc"
    "github.com/tandatle266/gotel/grpc/tracing"
    "github.com/grpc-ecosystem/go-grpc-middleware/recovery" // Optional recovery
)

opts := tracing.InstrumentGRPCServerOptions(
    grpc.UnaryInterceptor(recovery.UnaryServerInterceptor()), // Optional: your other interceptors
)
grpcServer := grpc.NewServer(opts...)

// Register services
pb.RegisterYourServiceServer(grpcServer, yourServiceImplementation)

lis, err := net.Listen("tcp", ":50051")
if err != nil {
    log.Fatal(err)
}
if err := grpcServer.Serve(lis); err != nil {
    log.Fatal(err)
}
```

---

### 3. Instrument a Gin HTTP Server

Add tracing middleware for your Gin server:

```go
import (
    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

r := gin.Default()
r.Use(otelgin.Middleware("your-gin-service")) // Or wrap this in your own helper
// ...routes...
r.Run(":8080")
```

---

### 4. Instrument a Redis Client

Instrument a go-redis client with OpenTelemetry:

```go
import (
    "github.com/redis/go-redis/v9"
    "github.com/tandatle266/gotel/redis/tracing"
)

rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
    // ...other options
})

// Just one line to add tracing
if err := tracing.InstrumentRedisClient(rdb); err != nil {
    log.Fatal(err)
}

// Use rdb as usual
```

---

### 5. Example Folder

See the [`example/`](./example/) folder for ready-to-run code using gRPC, Gin, and Redis with tracing.

---

## Summary

* **gRPC client/server:** One line to enable OpenTelemetry distributed tracing.
* **Gin:** Minimal setup for HTTP tracing with full context propagation.
* **Redis:** Instrument go-redis with a single function call.
* **No code changes to your application logic.**
* **Works with OpenTelemetry Collector, Jaeger, Tempo, and more.**

---

## Contributing

Contributions and suggestions are welcome!
Please open issues or submit pull requests.

---

## License

MIT License
