package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Get user ID from context if available
		var userID string
		if param.Keys != nil {
			if uid, exists := param.Keys["user_id"]; exists {
				if id, ok := uid.(uuid.UUID); ok {
					userID = id.String()
				}
			}
		}

		// Log request
		logger.Info("HTTP Request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
			zap.String("user_id", userID),
			zap.Int("body_size", param.BodySize),
			zap.String("error", param.ErrorMessage),
		)

		return ""
	})
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// DetailedLoggingMiddleware provides detailed request/response logging
func DetailedLoggingMiddleware(logger *zap.Logger, logRequestBody bool, logResponseBody bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get request ID
		requestID, _ := c.Get("request_id")
		reqID, _ := requestID.(string)

		// Create logger with request ID
		reqLogger := logger.With(zap.String("request_id", reqID))

		// Log request body if enabled
		var requestBody []byte
		if logRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Capture response body if enabled
		var responseBody *bytes.Buffer
		var responseWriter gin.ResponseWriter
		if logResponseBody {
			responseBody = new(bytes.Buffer)
			responseWriter = &responseBodyWriter{
				ResponseWriter: c.Writer,
				body:           responseBody,
			}
			c.Writer = responseWriter
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get user info from context
		var userID, userEmail string
		if uid, exists := c.Get("user_id"); exists {
			if id, ok := uid.(uuid.UUID); ok {
				userID = id.String()
			}
		}
		if email, exists := c.Get("user_email"); exists {
			if e, ok := email.(string); ok {
				userEmail = e
			}
		}

		// Build log fields
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", raw),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("user_id", userID),
			zap.String("user_email", userEmail),
			zap.Int("response_size", c.Writer.Size()),
		}

		// Add request body if logged
		if logRequestBody && len(requestBody) > 0 {
			fields = append(fields, zap.String("request_body", string(requestBody)))
		}

		// Add response body if logged
		if logResponseBody && responseBody != nil && responseBody.Len() > 0 {
			fields = append(fields, zap.String("response_body", responseBody.String()))
		}

		// Add errors if any
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log based on status code
		if c.Writer.Status() >= 500 {
			reqLogger.Error("HTTP Request - Server Error", fields...)
		} else if c.Writer.Status() >= 400 {
			reqLogger.Warn("HTTP Request - Client Error", fields...)
		} else {
			reqLogger.Info("HTTP Request - Success", fields...)
		}
	}
}

// responseBodyWriter captures response body for logging
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Only add HSTS in production with HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// RecoveryMiddleware provides panic recovery with logging
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Get request ID
		requestID, _ := c.Get("request_id")
		reqID, _ := requestID.(string)

		// Get user info
		var userID string
		if uid, exists := c.Get("user_id"); exists {
			if id, ok := uid.(uuid.UUID); ok {
				userID = id.String()
			}
		}

		logger.Error("Panic recovered",
			zap.String("request_id", reqID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_id", userID),
			zap.Any("error", recovered),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		c.JSON(500, gin.H{
			"error":      "Internal server error",
			"request_id": reqID,
		})
	})
}

// RateLimitInfo stores rate limit information
type RateLimitInfo struct {
	requests map[string][]time.Time
	maxReqs  int
	window   time.Duration
}

// NewRateLimit creates a new rate limiter
func NewRateLimit(maxRequests int, window time.Duration) *RateLimitInfo {
	return &RateLimitInfo{
		requests: make(map[string][]time.Time),
		maxReqs:  maxRequests,
		window:   window,
	}
}

// RateLimitMiddleware implements basic rate limiting
func RateLimitMiddleware(rateLimiter *RateLimitInfo, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// Clean old requests
		if requests, exists := rateLimiter.requests[clientIP]; exists {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < rateLimiter.window {
					validRequests = append(validRequests, reqTime)
				}
			}
			rateLimiter.requests[clientIP] = validRequests
		}

		// Check rate limit
		requests := rateLimiter.requests[clientIP]
		if len(requests) >= rateLimiter.maxReqs {
			logger.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.Int("requests", len(requests)),
				zap.Int("max_requests", rateLimiter.maxReqs),
				zap.Duration("window", rateLimiter.window),
			)

			c.Header("X-RateLimit-Limit", string(rune(rateLimiter.maxReqs)))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", string(rune(now.Add(rateLimiter.window).Unix())))

			c.JSON(429, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		// Add current request
		rateLimiter.requests[clientIP] = append(requests, now)

		// Set rate limit headers
		remaining := rateLimiter.maxReqs - len(rateLimiter.requests[clientIP])
		c.Header("X-RateLimit-Limit", string(rune(rateLimiter.maxReqs)))
		c.Header("X-RateLimit-Remaining", string(rune(remaining)))
		c.Header("X-RateLimit-Reset", string(rune(now.Add(rateLimiter.window).Unix())))

		c.Next()
	}
}
