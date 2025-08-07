package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func GinMiddleware(serviceName string) gin.HandlerFunc {
	mw := otelgin.Middleware(serviceName,
		otelgin.WithTracerProvider(otel.GetTracerProvider()),
		otelgin.WithSpanStartOptions(
			trace.WithAttributes(
				attribute.String("service.name", serviceName),
				attribute.String("env", "production"),
			),
		),
		otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
			return fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		}),
	)

	return func(c *gin.Context) {
		mw(c)

		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() && c.Writer.Status() >= http.StatusBadRequest {
			span.SetStatus(codes.Error, http.StatusText(c.Writer.Status()))
			span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
		}
	}
}

func InstrumentGinEngine(engine *gin.Engine, serviceName string) {
    engine.Use(otelgin.Middleware(serviceName))
}
