package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	gotelGrpc "github.com/tandatle266/gotel/grpc"
	gotelGin "github.com/tandatle266/gotel/middleware"
	gotelRedis "github.com/tandatle266/gotel/redis"

	"google.golang.org/grpc"

	pb "github.com/tandatle266/gotel/example/proto"
)

// Dummy implementation
type server struct {
    pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
    opts := gotelGrpc.InstrumentGRPCServerOptions(
        // grpc.UnaryInterceptor(recovery.UnaryServerInterceptor()), // interceptor other (recovery/log...)
    )

    grpcServer := grpc.NewServer(opts...)
    pb.RegisterGreeterServer(grpcServer, &server{})

    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    fmt.Println("gRPC server listening at :50051")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }

    // gin server
    r := gin.Default()

    r.Use(gotelGin.GinMiddleware("gin-server-demo"))

    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    r.Run(":8080")


    // redis
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    if err := gotelRedis.InstrumentRedisClient(rdb); err != nil {
        log.Fatalf("Could not instrument redis: %v", err)
    }

    ctx := context.Background()
    if err := rdb.Set(ctx, "hello", "world", 0).Err(); err != nil {
        log.Fatalf("Could not set value: %v", err)
    }

    val, err := rdb.Get(ctx, "hello").Result()
    if err != nil {
        log.Fatalf("Could not get value: %v", err)
    }
    fmt.Println("Value:", val)
}
