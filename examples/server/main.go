package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	httpPort = ":8080"
	grpcPort = ":9090"
	serviceName = "gotel-example-server"
)

// User model for database operations
type User struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Server struct contains all dependencies
type Server struct {
	db     *gorm.DB
	redis  *redis.Client
	tracer trace.Tracer
}

// NewServer creates a new server instance
func NewServer() *Server {
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
		log.Println("🛑 Shutting down...")
		shutdown()
		os.Exit(0)
	}()

	// Initialize database with tracing
	db, err := tracing.NewTracedDatabase(tracing.DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "gotel",
		Password: "gotel123",
		DBName:   "gotel_examples",
		Schema:   "public",
	}, serviceName)
	if err != nil {
		log.Printf("⚠️  Database connection failed: %v", err)
		log.Println("💡 Make sure PostgreSQL is running: docker-compose up postgres")
	} else {
		// Auto-migrate user table
		db.AutoMigrate(&User{})
		log.Println("✅ Database connected with tracing")
	}

	// Initialize Redis with tracing
	redisClient := tracing.NewTracedRedisClient(tracing.RedisConfig{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
	}, serviceName)

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("⚠️  Redis connection failed: %v", err)
		log.Println("💡 Make sure Redis is running: docker-compose up redis")
	} else {
		log.Println("✅ Redis connected with tracing")
	}

	return &Server{
		db:     db,
		redis:  redisClient,
		tracer: otel.Tracer(serviceName),
	}
}

// HTTP Handlers
func (s *Server) setupHTTPRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	// Add tracing middleware
	router.Use(tracing.HTTPMiddleware(serviceName))
	router.Use(gin.Recovery())
	
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": serviceName,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Example endpoints
	api := router.Group("/api/v1")
	{
		api.GET("/hello/:name", s.handleHello)
		api.GET("/users/:id", s.handleGetUser)
		api.POST("/users", s.handleCreateUser)
		api.GET("/cache/:key", s.handleGetFromCache)
		api.POST("/cache", s.handleSetToCache)
		api.GET("/fail/:rate", s.handleFailSometimes)
	}

	return router
}

func (s *Server) handleHello(c *gin.Context) {
	name := c.Param("name")
	
	// Add custom attributes to span
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attribute.String("user.name", name))
	
	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Hello, %s!", name),
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   serviceName,
	})
}

func (s *Server) handleGetUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")
	
	// Create child span
	ctx, span := s.tracer.Start(ctx, "get_user_handler")
	defer span.End()
	
	span.SetAttributes(attribute.String("user.id", userID))
	
	if s.db == nil {
		span.SetAttributes(attribute.String("error", "database not available"))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}
	
	var user User
	result := s.db.WithContext(ctx).First(&user, "id = ?", userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		span.RecordError(result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	
	c.JSON(http.StatusOK, user)
}

func (s *Server) handleCreateUser(c *gin.Context) {
	ctx := c.Request.Context()
	
	ctx, span := s.tracer.Start(ctx, "create_user_handler")
	defer span.End()
	
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}
	
	user.CreatedAt = time.Now()
	result := s.db.WithContext(ctx).Create(&user)
	if result.Error != nil {
		span.RecordError(result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	
	span.SetAttributes(
		attribute.Int64("user.id", user.ID),
		attribute.String("user.name", user.Name),
	)
	
	c.JSON(http.StatusCreated, user)
}

func (s *Server) handleGetFromCache(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")
	
	ctx, span := s.tracer.Start(ctx, "get_from_cache")
	defer span.End()
	
	span.SetAttributes(attribute.String("cache.key", key))
	
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		span.SetAttributes(attribute.Bool("cache.hit", false))
		c.JSON(http.StatusNotFound, gin.H{
			"key":       key,
			"cache_hit": false,
			"message":   "Key not found in cache",
		})
		return
	} else if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}
	
	span.SetAttributes(attribute.Bool("cache.hit", true))
	c.JSON(http.StatusOK, gin.H{
		"key":       key,
		"value":     val,
		"cache_hit": true,
	})
}

func (s *Server) handleSetToCache(c *gin.Context) {
	ctx := c.Request.Context()
	
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
		TTL   int    `json:"ttl"` // seconds
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx, span := s.tracer.Start(ctx, "set_to_cache")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("cache.key", req.Key),
		attribute.Int("cache.ttl", req.TTL),
	)
	
	ttl := time.Duration(req.TTL) * time.Second
	if req.TTL == 0 {
		ttl = time.Hour // default 1 hour
	}
	
	err := s.redis.Set(ctx, req.Key, req.Value, ttl).Err()
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"key":     req.Key,
		"value":   req.Value,
		"ttl":     int(ttl.Seconds()),
		"message": "Cached successfully",
	})
}

func (s *Server) handleFailSometimes(c *gin.Context) {
	ctx := c.Request.Context()
	rateStr := c.Param("rate")
	
	ctx, span := s.tracer.Start(ctx, "fail_sometimes")
	defer span.End()
	
	// Parse fail rate (0.0 to 1.0)
	var rate float64 = 0.5 // default 50%
	if rateStr != "" {
		if parsed, err := time.ParseDuration(rateStr + "ms"); err == nil {
			rate = float64(parsed.Milliseconds()) / 1000.0
		}
	}
	
	span.SetAttributes(attribute.Float64("fail.rate", rate))
	
	// Simulate random failure
	if rand.Float64() < rate {
		err := fmt.Errorf("simulated failure (rate: %.2f)", rate)
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "Simulated failure",
			"fail_rate": rate,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"result":    "success",
		"fail_rate": rate,
		"message":   "Operation completed successfully",
	})
}

// gRPC Service Implementation
func (s *Server) SayHello(ctx context.Context, req *proto.HelloRequest) (*proto.HelloResponse, error) {
	ctx, span := s.tracer.Start(ctx, "SayHello")
	defer span.End()
	
	span.SetAttributes(attribute.String("request.name", req.Name))
	
	return &proto.HelloResponse{
		Message:   fmt.Sprintf("Hello, %s! (from gRPC)", req.Name),
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Server) GetUser(ctx context.Context, req *proto.GetUserRequest) (*proto.GetUserResponse, error) {
	ctx, span := s.tracer.Start(ctx, "GetUser")
	defer span.End()
	
	span.SetAttributes(attribute.Int64("user.id", req.UserId))
	
	if s.db == nil {
		return nil, status.Error(codes.Unavailable, "Database not available")
	}
	
	var user User
	result := s.db.WithContext(ctx).First(&user, "id = ?", req.UserId)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		span.RecordError(result.Error)
		return nil, status.Error(codes.Internal, "Database error")
	}
	
	return &proto.GetUserResponse{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) GetCachedData(ctx context.Context, req *proto.CacheRequest) (*proto.CacheResponse, error) {
	ctx, span := s.tracer.Start(ctx, "GetCachedData")
	defer span.End()
	
	span.SetAttributes(attribute.String("cache.key", req.Key))
	
	val, err := s.redis.Get(ctx, req.Key).Result()
	if err == redis.Nil {
		span.SetAttributes(attribute.Bool("cache.hit", false))
		return &proto.CacheResponse{
			Value:    "",
			CacheHit: false,
		}, nil
	} else if err != nil {
		span.RecordError(err)
		return nil, status.Error(codes.Internal, "Redis error")
	}
	
	span.SetAttributes(attribute.Bool("cache.hit", true))
	return &proto.CacheResponse{
		Value:    val,
		CacheHit: true,
	}, nil
}

func (s *Server) FailSometimes(ctx context.Context, req *proto.FailRequest) (*proto.FailResponse, error) {
	ctx, span := s.tracer.Start(ctx, "FailSometimes")
	defer span.End()
	
	span.SetAttributes(attribute.Float64("fail.rate", float64(req.FailRate)))
	
	if rand.Float32() < req.FailRate {
		err := fmt.Errorf("simulated gRPC failure (rate: %.2f)", req.FailRate)
		span.RecordError(err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	
	return &proto.FailResponse{
		Result: "gRPC operation completed successfully",
	}, nil
}

func main() {
	log.SetPrefix("🚀 [GOTEL-SERVER] ")
	log.Println("Starting Gotel Example Server...")
	
	server := NewServer()
	
	// Setup HTTP server
	httpRouter := server.setupHTTPRoutes()
	httpServer := &http.Server{
		Addr:    httpPort,
		Handler: httpRouter,
	}
	
	// Setup gRPC server
	grpcServer := grpc.NewServer(tracing.ServerOptions()...)
	proto.RegisterExampleServiceServer(grpcServer, server)
	
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}
	
	// Start servers
	var wg sync.WaitGroup
	
	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("🌐 HTTP server starting on %s", httpPort)
		log.Printf("📊 Health check: http://localhost%s/health", httpPort)
		log.Printf("🔗 API docs: http://localhost%s/api/v1/hello/world", httpPort)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
	
	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("🔌 gRPC server starting on %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()
	
	log.Println("✅ All servers started successfully!")
	log.Println("📈 View traces at: http://localhost:16686")
	log.Println("🛑 Press Ctrl+C to shutdown")
	
	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	
	log.Println("🛑 Shutting down servers...")
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	httpServer.Shutdown(ctx)
	grpcServer.GracefulStop()
	
	wg.Wait()
	log.Println("👋 Server shutdown complete")
}