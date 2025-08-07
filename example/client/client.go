package client

import (
	"context"
	"fmt"
	"log"
	"time"

	gotelGrpc "github.com/tandatle266/gotel/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/tandatle266/gotel/example/proto"
)

func main() {
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()), // optional TLS
    }

    opts = gotelGrpc.InstrumentGRPCDialOptions(opts...)

    conn, err := grpc.Dial("localhost:50051", opts...)
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    c := pb.NewGreeterClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    r, err := c.SayHello(ctx, &pb.HelloRequest{Name: "OpenAI"})
    if err != nil {
        log.Fatalf("could not greet: %v", err)
    }
    fmt.Printf("Greeting: %s\n", r.GetMessage())
}
