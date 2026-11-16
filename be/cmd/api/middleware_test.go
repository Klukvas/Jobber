package main

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/andreypavlenko/jobber/modules/admin"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type adminIdentity struct{}

func (adminIdentity) GetByID(_ context.Context, id string) (*userModel.User, error) {
	return &userModel.User{ID: id, Email: "ordinary@example.com", EmailVerified: true}, nil
}

func TestAdminCacheControlWithGlobalMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerGlobalMiddleware(router, &logger.Logger{Logger: zap.NewNop()}, false, "http://localhost:3001")
	jwt := auth.NewJWTManager("fixture-access", "fixture-refresh", time.Hour, time.Hour)
	token, err := jwt.GenerateAccessToken("00000000-0000-4000-8000-000000000001")
	require.NoError(t, err)
	admin.NewHandler(nil, adminIdentity{}, "admin@example.com", "http://localhost:3001").
		RegisterRoutes(router.Group("/api/v1"), auth.AuthMiddleware(jwt))
	router.GET("/api/v1/jobs", func(c *gin.Context) { c.Status(200) })
	router.GET("/api/v1/administrator", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=60")
		c.Status(200)
	})
	handler := withAdminNoStore(router)

	for _, test := range []struct {
		name, method, path string
		authenticated      bool
		status             int
	}{
		{"capability", "GET", "/api/v1/admin/access", true, 200},
		{"capability trailing slash redirect", "GET", "/api/v1/admin/access/", false, 301},
		{"users trailing slash redirect", "GET", "/api/v1/admin/users/", false, 301},
		{"invalid UUID before authentication", "GET", "/api/v1/admin/users/invalid", false, 400},
		{"invalid UUID with session", "GET", "/api/v1/admin/users/invalid", true, 400},
		{"invalid grant", "POST", "/api/v1/admin/users/00000000-0000-4000-8000-000000000001/manual-access/invalid/revoke", true, 400},
		{"unauthenticated", "GET", "/api/v1/admin/users", false, 401},
		{"forbidden", "GET", "/api/v1/admin/users", true, 403},
		{"preflight before authentication", "OPTIONS", "/api/v1/admin/users", false, 204},
		{"unknown admin route", "GET", "/api/v1/admin/missing", false, 404},
		{"admin root", "GET", "/api/v1/admin", false, 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			if test.authenticated {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			require.Equal(t, test.status, response.Code, response.Body.String())
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			if test.status == 301 {
				require.Equal(t, test.path[:len(test.path)-1], response.Header().Get("Location"))
			}
		})
	}
	for _, test := range []struct {
		path, cacheControl string
		status             int
	}{
		{"/api/v1/jobs", "", 200},
		{"/api/v1/jobs/", "", 301},
		{"/api/v1/administrator", "public, max-age=60", 200},
	} {
		t.Run("non-admin "+test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("GET", test.path, nil))
			require.Equal(t, test.status, response.Code)
			require.Equal(t, test.cacheControl, response.Header().Get("Cache-Control"))
		})
	}
}
