package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthMiddleware(t *testing.T) {
	jwtManager := NewJWTManager("access-secret-32-characters!!", "refresh-secret-32-characters!", 15*time.Minute, 7*24*time.Hour)

	t.Run("allows request with valid token", func(t *testing.T) {
		userID := "user-123"
		token, _ := jwtManager.GenerateAccessToken(userID)

		router := setupTestRouter()
		router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
			uid, _ := GetUserID(c)
			c.JSON(http.StatusOK, gin.H{"user_id": uid})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects request without authorization header", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects request with invalid authorization format", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects request with non-Bearer prefix", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic sometoken")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects request with invalid token", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects request with expired token", func(t *testing.T) {
		// Create a JWT manager with expired tokens
		expiredJwt := NewJWTManager("access-secret-32-characters!!", "refresh-secret-32-characters!", -1*time.Second, 7*24*time.Hour)
		token, _ := expiredJwt.GenerateAccessToken("user-123")

		router := setupTestRouter()
		router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("returns user ID when set", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", "user-123")

		userID, exists := GetUserID(c)

		assert.True(t, exists)
		assert.Equal(t, "user-123", userID)
	})

	t.Run("returns false when user ID not set", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID, exists := GetUserID(c)

		assert.False(t, exists)
		assert.Empty(t, userID)
	})
}

func TestMustGetUserID(t *testing.T) {
	t.Run("returns user ID when set", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", "user-123")

		userID, ok := MustGetUserID(c)

		assert.True(t, ok)
		assert.Equal(t, "user-123", userID)
	})

	t.Run("returns error response when user ID not set", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID, ok := MustGetUserID(c)

		assert.False(t, ok)
		assert.Empty(t, userID)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
