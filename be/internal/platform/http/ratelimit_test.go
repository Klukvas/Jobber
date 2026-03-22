package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupRateLimitTest(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	cfg := RateLimitConfig{
		MaxRequests: 3,
		Window:      1 * time.Minute,
		KeyPrefix:   "test",
	}

	t.Run("allows requests under limit", func(t *testing.T) {
		_, rdb := setupRateLimitTest(t)

		router := gin.New()
		router.Use(RateLimitMiddleware(rdb, cfg, logger))
		router.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		for i := range cfg.MaxRequests {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		_, rdb := setupRateLimitTest(t)

		router := gin.New()
		router.Use(RateLimitMiddleware(rdb, cfg, logger))
		router.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Exhaust the limit
		for range cfg.MaxRequests {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
		}

		// Next request should be blocked
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("different IPs have independent limits", func(t *testing.T) {
		_, rdb := setupRateLimitTest(t)

		router := gin.New()
		router.Use(RateLimitMiddleware(rdb, cfg, logger))
		router.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Exhaust limit for IP1
		for range cfg.MaxRequests {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
		}

		// IP2 should still be allowed
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "10.0.0.2:5678"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	cfg := RateLimitConfig{
		MaxRequests: 2,
		Window:      1 * time.Minute,
		KeyPrefix:   "user-test",
	}

	t.Run("uses user_id when present in context", func(t *testing.T) {
		mr, rdb := setupRateLimitTest(t)

		router := gin.New()
		// Simulate auth middleware setting user_id
		router.Use(func(c *gin.Context) {
			c.Set("user_id", "user-abc")
			c.Next()
		})
		router.Use(UserRateLimitMiddleware(rdb, cfg, logger))
		router.GET("/api", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// Exhaust limit
		for range cfg.MaxRequests {
			req := httptest.NewRequest(http.MethodGet, "/api", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
		}

		// Next request should be blocked
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		// Verify the key used contains user_id
		keys := mr.Keys()
		found := false
		for _, k := range keys {
			if k == "ratelimit:user-test:user:user-abc" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected Redis key with user_id, got keys: %v", keys)
	})

	t.Run("falls back to IP when no user_id", func(t *testing.T) {
		mr, rdb := setupRateLimitTest(t)

		router := gin.New()
		// No auth middleware — user_id not set
		router.Use(UserRateLimitMiddleware(rdb, cfg, logger))
		router.GET("/api", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.RemoteAddr = "172.16.0.1:9999"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the key used contains the IP, not "user:"
		keys := mr.Keys()
		found := false
		for _, k := range keys {
			if k == "ratelimit:user-test:172.16.0.1" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected Redis key with IP fallback, got keys: %v", keys)
	})

	t.Run("safe type assertion on non-string user_id", func(t *testing.T) {
		mr, rdb := setupRateLimitTest(t)

		router := gin.New()
		// Set user_id as non-string type
		router.Use(func(c *gin.Context) {
			c.Set("user_id", 12345)
			c.Next()
		})
		router.Use(UserRateLimitMiddleware(rdb, cfg, logger))
		router.GET("/api", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.RemoteAddr = "10.0.0.5:4321"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Should fall back to IP since type assertion to string fails
		keys := mr.Keys()
		found := false
		for _, k := range keys {
			if k == "ratelimit:user-test:10.0.0.5" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected Redis key with IP fallback on non-string user_id, got keys: %v", keys)
	})
}
