package autoloadx

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

var headerXRequestID string

// RequestIDOption for request id generator
type RequestIDOption func(*requestIDConfig)

type (
	RequestIDGenerator func(ctx context.Context, c *app.RequestContext) string
	RequestIDHandler   func(ctx context.Context, c *app.RequestContext, requestID string)
)

// RequestIDNew initializes the RequestID middleware.
func RequestIDNew(opts ...RequestIDOption) app.HandlerFunc {
	cfg := &requestIDConfig{
		generator: func(ctx context.Context, c *app.RequestContext) string {
			return uuid.New().String()
		},
		headerKey: "X-Request-ID",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx context.Context, c *app.RequestContext) {
		// Get id from request
		rid := c.Request.Header.Get(string(cfg.headerKey))
		if rid == "" {
			rid = cfg.generator(ctx, c)
		}
		headerXRequestID = string(cfg.headerKey)
		if cfg.handler != nil {
			cfg.handler(ctx, c, rid)
		}
		// Set the id to ensure that the request id is in the response
		c.Header(headerXRequestID, rid)
		ctx = context.WithValue(ctx, headerXRequestID, rid)
		c.QueryArgs().Add(headerXRequestID, rid)
		c.Next(ctx)
	}
}

type RequestIDHeaderStrKey string

// WithGenerator set generator function
func WithGenerator(g RequestIDGenerator) RequestIDOption {
	return func(cfg *requestIDConfig) {
		cfg.generator = g
	}
}

// WithCustomHeaderStrKey set custom header key for request id
func WithCustomHeaderStrKey(s RequestIDHeaderStrKey) RequestIDOption {
	return func(cfg *requestIDConfig) {
		cfg.headerKey = s
	}
}

// WithHandler set handler function for request id with context
func WithHandler(handler RequestIDHandler) RequestIDOption {
	return func(cfg *requestIDConfig) {
		cfg.handler = handler
	}
}

// Config defines the config for RequestID middleware
type requestIDConfig struct {
	// Generator defines a function to generate an ID.
	// Optional. Default: func(ctx context.Context, c *app.RequestContext) string {
	//   return uuid.New().String()
	// }
	generator RequestIDGenerator
	headerKey RequestIDHeaderStrKey
	handler   RequestIDHandler
}

// RequestIDGet returns the request identifier
func RequestIDGet(c *app.RequestContext) string {
	return c.Response.Header.Get(headerXRequestID)
}
