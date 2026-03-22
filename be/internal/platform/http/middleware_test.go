package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorCode  string
		errorMsg   string
	}{
		{
			name:       "bad request error",
			statusCode: http.StatusBadRequest,
			errorCode:  "INVALID_INPUT",
			errorMsg:   "missing required field",
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			errorCode:  "INTERNAL_ERROR",
			errorMsg:   "something went wrong",
		},
		{
			name:       "not found error",
			statusCode: http.StatusNotFound,
			errorCode:  "NOT_FOUND",
			errorMsg:   "resource not found",
		},
		{
			name:       "unauthorized error",
			statusCode: http.StatusUnauthorized,
			errorCode:  "UNAUTHORIZED",
			errorMsg:   "invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			RespondWithError(c, tt.statusCode, tt.errorCode, tt.errorMsg)

			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var body ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, tt.errorCode, body.ErrorCode)
			assert.Equal(t, tt.errorMsg, body.ErrorMessage)
		})
	}
}

func TestRespondWithSuccess(t *testing.T) {
	t.Run("with data wraps in Data field", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		payload := map[string]string{"key": "value"}
		RespondWithSuccess(c, http.StatusOK, payload)

		assert.Equal(t, http.StatusOK, w.Code)

		var body SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.NotNil(t, body.Data)

		dataMap, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", dataMap["key"])
	})

	t.Run("with nil data returns empty JSON object", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		RespondWithSuccess(c, http.StatusOK, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var body map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Empty(t, body)
	})

	t.Run("with created status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		RespondWithSuccess(c, http.StatusCreated, map[string]int{"id": 42})

		assert.Equal(t, http.StatusCreated, w.Code)

		var body SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.NotNil(t, body.Data)
	})

	t.Run("with slice data", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		RespondWithSuccess(c, http.StatusOK, []string{"a", "b"})

		assert.Equal(t, http.StatusOK, w.Code)

		var body SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
	})
}

func TestRespondWithData(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       interface{}
	}{
		{
			name:       "map data",
			statusCode: http.StatusOK,
			data:       map[string]string{"hello": "world"},
		},
		{
			name:       "slice data",
			statusCode: http.StatusOK,
			data:       []string{"a", "b", "c"},
		},
		{
			name:       "struct data",
			statusCode: http.StatusOK,
			data:       ErrorResponse{ErrorCode: "X", ErrorMessage: "Y"},
		},
		{
			name:       "nested struct",
			statusCode: http.StatusAccepted,
			data:       HealthResponse{Status: "ok", Version: "1.0.0", Services: map[string]string{"db": "up"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			RespondWithData(c, tt.statusCode, tt.data)

			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var raw json.RawMessage
			err := json.Unmarshal(w.Body.Bytes(), &raw)
			require.NoError(t, err)
		})
	}
}

func TestRespondWithHealth(t *testing.T) {
	tests := []struct {
		name           string
		services       map[string]string
		expectedStatus string
	}{
		{
			name:           "all services up",
			services:       map[string]string{"db": "up", "redis": "up", "s3": "up"},
			expectedStatus: "healthy",
		},
		{
			name:           "one service down",
			services:       map[string]string{"db": "up", "redis": "down", "s3": "up"},
			expectedStatus: "degraded",
		},
		{
			name:           "all services down",
			services:       map[string]string{"db": "down", "redis": "down"},
			expectedStatus: "degraded",
		},
		{
			name:           "empty services map",
			services:       map[string]string{},
			expectedStatus: "healthy",
		},
		{
			name:           "single service up",
			services:       map[string]string{"db": "up"},
			expectedStatus: "healthy",
		},
		{
			name:           "service with unexpected status string",
			services:       map[string]string{"db": "unknown"},
			expectedStatus: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			RespondWithHealth(c, tt.services)

			assert.Equal(t, http.StatusOK, w.Code)

			var body HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, body.Status)
			assert.Equal(t, "1.0.0", body.Version)
			assert.Equal(t, tt.services, body.Services)
		})
	}
}

// ---------------------------------------------------------------------------
// CORSMiddleware
// ---------------------------------------------------------------------------

func TestCORSMiddleware(t *testing.T) {
	t.Run("wildcard origin sets Allow-Origin to star", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("*"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://anything.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("specific origin allowed", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("https://app.example.com,https://admin.example.com"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://app.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"))
	})

	t.Run("second allowed origin also works", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("https://app.example.com,https://admin.example.com"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://admin.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "https://admin.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("origin not allowed omits Allow-Origin", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("https://app.example.com"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("preflight OPTIONS returns 204 and aborts", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("*"))

		handlerCalled := false
		router.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://app.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.False(t, handlerCalled, "handler should not be called for OPTIONS preflight")
	})

	t.Run("sets required CORS headers", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("*"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
		assert.Contains(t, allowHeaders, "Authorization")
		assert.Contains(t, allowHeaders, "Content-Type")
		assert.Contains(t, allowHeaders, "X-Request-ID")

		allowMethods := w.Header().Get("Access-Control-Allow-Methods")
		assert.Contains(t, allowMethods, "POST")
		assert.Contains(t, allowMethods, "GET")
		assert.Contains(t, allowMethods, "PUT")
		assert.Contains(t, allowMethods, "PATCH")
		assert.Contains(t, allowMethods, "DELETE")
		assert.Contains(t, allowMethods, "OPTIONS")
	})

	t.Run("case-insensitive origin matching", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("https://App.Example.Com"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://app.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("no origin header with specific origins configured", func(t *testing.T) {
		router := gin.New()
		router.Use(CORSMiddleware("https://app.example.com"))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
}

// ---------------------------------------------------------------------------
// RequestIDMiddleware
// ---------------------------------------------------------------------------

func TestRequestIDMiddleware(t *testing.T) {
	t.Run("generates UUID when no header provided", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())

		var capturedID string
		router.GET("/test", func(c *gin.Context) {
			val, _ := c.Get("request_id")
			capturedID = val.(string)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		_, err := uuid.Parse(capturedID)
		assert.NoError(t, err, "generated request ID should be a valid UUID")
		assert.Equal(t, capturedID, w.Header().Get("X-Request-ID"))
	})

	t.Run("accepts valid client-supplied ID", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())

		var capturedID string
		router.GET("/test", func(c *gin.Context) {
			val, _ := c.Get("request_id")
			capturedID = val.(string)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		clientID := "my-custom-request-id-123"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", clientID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, clientID, capturedID)
		assert.Equal(t, clientID, w.Header().Get("X-Request-ID"))
	})

	t.Run("accepts UUID as client ID", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())

		var capturedID string
		router.GET("/test", func(c *gin.Context) {
			val, _ := c.Get("request_id")
			capturedID = val.(string)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		clientID := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", clientID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, clientID, capturedID)
	})

	t.Run("replaces invalid ID with special characters", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())

		var capturedID string
		router.GET("/test", func(c *gin.Context) {
			val, _ := c.Get("request_id")
			capturedID = val.(string)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "invalid<script>alert(1)</script>")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		_, err := uuid.Parse(capturedID)
		assert.NoError(t, err, "should generate a new UUID for invalid input")
	})

	t.Run("replaces ID exceeding max length", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())

		var capturedID string
		router.GET("/test", func(c *gin.Context) {
			val, _ := c.Get("request_id")
			capturedID = val.(string)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		longID := makeString('x', maxRequestIDLength+10)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", longID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEqual(t, longID, capturedID)
		_, err := uuid.Parse(capturedID)
		assert.NoError(t, err)
	})

	t.Run("calls Next", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())

		called := false
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.True(t, called, "handler should have been called")
	})
}

// ---------------------------------------------------------------------------
// isValidRequestID
// ---------------------------------------------------------------------------

func TestIsValidRequestID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{name: "simple alphanumeric", id: "abc123", expected: true},
		{name: "with dashes", id: "req-id-123", expected: true},
		{name: "with underscores", id: "req_id_123", expected: true},
		{name: "with dots", id: "req.id.123", expected: true},
		{name: "UUID format", id: "550e8400-e29b-41d4-a716-446655440000", expected: true},
		{name: "empty string", id: "", expected: false},
		{name: "with spaces", id: "req id", expected: false},
		{name: "with slashes", id: "req/id", expected: false},
		{name: "with angle brackets", id: "<script>", expected: false},
		{name: "with newlines", id: "req\nid", expected: false},
		{name: "at max length", id: makeString('a', maxRequestIDLength), expected: true},
		{name: "over max length", id: makeString('a', maxRequestIDLength+1), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidRequestID(tt.id))
		})
	}
}

// ---------------------------------------------------------------------------
// LoggerMiddleware
// ---------------------------------------------------------------------------

func TestLoggerMiddleware(t *testing.T) {
	t.Run("calls Next for 200 response", func(t *testing.T) {
		log := &logger.Logger{Logger: zap.NewNop()}

		router := gin.New()
		router.Use(RequestIDMiddleware())
		router.Use(LoggerMiddleware(log))

		handlerCalled := false
		router.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("handles 4xx status without panic", func(t *testing.T) {
		log := &logger.Logger{Logger: zap.NewNop()}

		router := gin.New()
		router.Use(RequestIDMiddleware())
		router.Use(LoggerMiddleware(log))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles 5xx status without panic", func(t *testing.T) {
		log := &logger.Logger{Logger: zap.NewNop()}

		router := gin.New()
		router.Use(RequestIDMiddleware())
		router.Use(LoggerMiddleware(log))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "fail"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles 3xx redirect status without panic", func(t *testing.T) {
		log := &logger.Logger{Logger: zap.NewNop()}

		router := gin.New()
		router.Use(RequestIDMiddleware())
		router.Use(LoggerMiddleware(log))
		router.GET("/test", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/new-location")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMovedPermanently, w.Code)
	})
}

// ---------------------------------------------------------------------------
// parseOrigins / isOriginAllowed
// ---------------------------------------------------------------------------

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected []string
	}{
		{name: "empty string", raw: "", expected: nil},
		{name: "wildcard", raw: "*", expected: nil},
		{name: "single origin", raw: "https://example.com", expected: []string{"https://example.com"}},
		{name: "multiple origins", raw: "https://a.com,https://b.com", expected: []string{"https://a.com", "https://b.com"}},
		{name: "with whitespace", raw: " https://a.com , https://b.com ", expected: []string{"https://a.com", "https://b.com"}},
		{name: "trailing comma produces no empty entry", raw: "https://a.com,", expected: []string{"https://a.com"}},
		{name: "leading comma produces no empty entry", raw: ",https://a.com", expected: []string{"https://a.com"}},
		{name: "three origins", raw: "https://a.com,https://b.com,https://c.com", expected: []string{"https://a.com", "https://b.com", "https://c.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOrigins(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  []string
		expected bool
	}{
		{name: "exact match", origin: "https://a.com", allowed: []string{"https://a.com"}, expected: true},
		{name: "case insensitive", origin: "https://A.COM", allowed: []string{"https://a.com"}, expected: true},
		{name: "not in list", origin: "https://evil.com", allowed: []string{"https://a.com"}, expected: false},
		{name: "empty allowed list", origin: "https://a.com", allowed: nil, expected: false},
		{name: "empty origin", origin: "", allowed: []string{"https://a.com"}, expected: false},
		{name: "match second in list", origin: "https://b.com", allowed: []string{"https://a.com", "https://b.com"}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isOriginAllowed(tt.origin, tt.allowed))
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeString(ch byte, length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}
