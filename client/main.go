package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serviceName = "gotel-example-client"
	serverHTTP  = "http://localhost:8080"
	serverGRPC  = "localhost:9090"
)

type Client struct {
	httpClient *http.Client
	grpcConn   *grpc.ClientConn
	grpcClient proto.ExampleServiceClient
	tracer     trace.Tracer
}

func NewClient() *Client {
	// Initialize tracing
	shutdown := tracing.InitTracer(tracing.Config{
		ServiceName: serviceName,
		Endpoint:    "localhost:4317",
		Insecure:    true,
		Environment: "development",
	})

	// Setup graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("🛑 Shutting down client...")
		shutdown()
		os.Exit(0)
	}()

	// Create HTTP client with tracing
	httpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   30 * time.Second,
	}

	// Create gRPC connection with tracing
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	dialOptions = append(dialOptions, tracing.ClientOptions()...)

	grpcConn, err := grpc.NewClient(serverGRPC, dialOptions...)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}

	grpcClient := proto.NewExampleServiceClient(grpcConn)

	return &Client{
		httpClient: httpClient,
		grpcConn:   grpcConn,
		grpcClient: grpcClient,
		tracer:     otel.Tracer(serviceName),
	}
}

func (c *Client) Close() {
	if c.grpcConn != nil {
		c.grpcConn.Close()
	}
}

// HTTP Client methods
func (c *Client) makeHTTPRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	ctx, span := c.tracer.Start(ctx, fmt.Sprintf("HTTP %s %s", method, path))
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", method),
		attribute.String("http.url", serverHTTP+path),
	)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, serverHTTP+path, reqBody)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	return resp, nil
}

func (c *Client) TestHTTPEndpoints(ctx context.Context) {
	log.Println("🌐 Testing HTTP endpoints...")

	// Test health check
	if resp, err := c.makeHTTPRequest(ctx, "GET", "/health", nil); err == nil {
		defer resp.Body.Close()
		log.Printf("✅ Health check: %d", resp.StatusCode)
	} else {
		log.Printf("❌ Health check failed: %v", err)
	}

	// Test hello endpoint
	if resp, err := c.makeHTTPRequest(ctx, "GET", "/api/v1/hello/Alice", nil); err == nil {
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		log.Printf("✅ Hello endpoint: %s", result["message"])
	} else {
		log.Printf("❌ Hello endpoint failed: %v", err)
	}

	// Test create user
	user := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}
	if resp, err := c.makeHTTPRequest(ctx, "POST", "/api/v1/users", user); err == nil {
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		if resp.StatusCode == 201 {
			log.Printf("✅ User created: ID %v", result["id"])
			
			// Test get user
			userID := fmt.Sprintf("%.0f", result["id"].(float64))
			if resp2, err2 := c.makeHTTPRequest(ctx, "GET", "/api/v1/users/"+userID, nil); err2 == nil {
				defer resp2.Body.Close()
				var user map[string]interface{}
				json.NewDecoder(resp2.Body).Decode(&user)
				log.Printf("✅ User retrieved: %s", user["name"])
			}
		} else {
			log.Printf("⚠️  User creation: %d (DB may not be available)", resp.StatusCode)
		}
	} else {
		log.Printf("❌ Create user failed: %v", err)
	}

	// Test cache operations
	cacheData := map[string]interface{}{
		"key":   "test-key",
		"value": "test-value",
		"ttl":   60,
	}
	if resp, err := c.makeHTTPRequest(ctx, "POST", "/api/v1/cache", cacheData); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			log.Println("✅ Cache set successfully")
			
			// Test get from cache
			if resp2, err2 := c.makeHTTPRequest(ctx, "GET", "/api/v1/cache/test-key", nil); err2 == nil {
				defer resp2.Body.Close()
				var result map[string]interface{}
				json.NewDecoder(resp2.Body).Decode(&result)
				log.Printf("✅ Cache retrieved: hit=%t", result["cache_hit"])
			}
		} else {
			log.Printf("⚠️  Cache set: %d (Redis may not be available)", resp.StatusCode)
		}
	} else {
		log.Printf("❌ Cache set failed: %v", err)
	}

	// Test failure simulation
	if resp, err := c.makeHTTPRequest(ctx, "GET", "/api/v1/fail/0.2", nil); err == nil {
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		if resp.StatusCode == 200 {
			log.Printf("✅ Failure test: %s", result["result"])
		} else {
			log.Printf("⚠️  Failure test: Simulated failure occurred")
		}
	}
}

// gRPC Client methods
func (c *Client) TestGRPCEndpoints(ctx context.Context) {
	log.Println("🔌 Testing gRPC endpoints...")

	// Test SayHello
	if resp, err := c.grpcClient.SayHello(ctx, &proto.HelloRequest{
		Name: "Bob",
	}); err == nil {
		log.Printf("✅ gRPC Hello: %s", resp.Message)
	} else {
		log.Printf("❌ gRPC Hello failed: %v", err)
	}

	// Test GetUser
	if resp, err := c.grpcClient.GetUser(ctx, &proto.GetUserRequest{
		UserId: 1,
	}); err == nil {
		log.Printf("✅ gRPC GetUser: %s (%s)", resp.Name, resp.Email)
	} else {
		log.Printf("⚠️  gRPC GetUser: %v (User may not exist)", err)
	}

	// Test GetCachedData
	if resp, err := c.grpcClient.GetCachedData(ctx, &proto.CacheRequest{
		Key: "test-key",
	}); err == nil {
		log.Printf("✅ gRPC Cache: hit=%t, value=%s", resp.CacheHit, resp.Value)
	} else {
		log.Printf("❌ gRPC Cache failed: %v", err)
	}

	// Test FailSometimes
	if resp, err := c.grpcClient.FailSometimes(ctx, &proto.FailRequest{
		FailRate: 0.3,
	}); err == nil {
		log.Printf("✅ gRPC Fail test: %s", resp.Result)
	} else {
		log.Printf("⚠️  gRPC Fail test: %v (Simulated failure)", err)
	}
}

// Comprehensive test with distributed tracing
func (c *Client) RunComprehensiveTest(ctx context.Context) {
	ctx, span := c.tracer.Start(ctx, "comprehensive_test")
	defer span.End()

	log.Println("🧪 Running comprehensive test with distributed tracing...")

	// Create a user via HTTP
	ctx, createSpan := c.tracer.Start(ctx, "create_user_flow")
	user := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
	}
	
	var userID string
	if resp, err := c.makeHTTPRequest(ctx, "POST", "/api/v1/users", user); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 201 {
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			userID = fmt.Sprintf("%.0f", result["id"].(float64))
			createSpan.SetAttributes(attribute.String("user.id", userID))
			log.Printf("✅ Created user via HTTP: ID %s", userID)
		}
	}
	createSpan.End()

	// Retrieve user via gRPC
	if userID != "" {
		ctx, getSpan := c.tracer.Start(ctx, "get_user_flow")
		if userID == "1" { // Fallback to ID 1 if creation failed
			if resp, err := c.grpcClient.GetUser(ctx, &proto.GetUserRequest{
				UserId: 1,
			}); err == nil {
				getSpan.SetAttributes(attribute.String("user.name", resp.Name))
				log.Printf("✅ Retrieved user via gRPC: %s", resp.Name)
			} else {
				log.Printf("⚠️  Failed to retrieve user via gRPC: %v", err)
			}
		}
		getSpan.End()
	}

	// Test cache via both HTTP and gRPC
	ctx, cacheSpan := c.tracer.Start(ctx, "cache_test_flow")
	
	// Set cache via HTTP
	cacheData := map[string]interface{}{
		"key":   "distributed-test",
		"value": "Hello from distributed tracing!",
		"ttl":   300,
	}
	c.makeHTTPRequest(ctx, "POST", "/api/v1/cache", cacheData)
	
	// Get cache via gRPC
	if resp, err := c.grpcClient.GetCachedData(ctx, &proto.CacheRequest{
		Key: "distributed-test",
	}); err == nil {
		cacheSpan.SetAttributes(
			attribute.Bool("cache.hit", resp.CacheHit),
			attribute.String("cache.value", resp.Value),
		)
		log.Printf("✅ Cache test complete: hit=%t", resp.CacheHit)
	}
	cacheSpan.End()

	log.Println("🎉 Comprehensive test completed!")
}

func main() {
	log.SetPrefix("🔍 [GOTEL-CLIENT] ")
	log.Println("Starting Gotel Example Client...")

	client := NewClient()
	defer client.Close()

	// Wait a moment for server to be ready
	time.Sleep(2 * time.Second)

	ctx := context.Background()

	// Run tests
	log.Println("🚀 Starting tests...")
	log.Println("📈 View traces at: http://localhost:16686")
	log.Println("")

	// Test HTTP endpoints
	client.TestHTTPEndpoints(ctx)
	log.Println("")

	// Test gRPC endpoints
	client.TestGRPCEndpoints(ctx)
	log.Println("")

	// Run comprehensive test
	client.RunComprehensiveTest(ctx)
	log.Println("")

	log.Println("✅ All tests completed!")
	log.Println("📊 Check Jaeger UI to see distributed traces across services")
	log.Println("🔗 Jaeger UI: http://localhost:16686")
	log.Println("")
	
	// Keep running for a bit to let traces export
	log.Println("⏳ Waiting 5 seconds for traces to export...")
	time.Sleep(5 * time.Second)
	
	log.Println("👋 Client finished")
}