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

	for _, test := range []struct {
		name, method, path string
		authenticated      bool
		status             int
	}{
		{"capability", "GET", "/api/v1/admin/access", true, 200},
		{"invalid UUID before authentication", "GET", "/api/v1/admin/users/invalid", false, 400},
		{"invalid UUID with session", "GET", "/api/v1/admin/users/invalid", true, 400},
		{"invalid grant", "POST", "/api/v1/admin/users/00000000-0000-4000-8000-000000000001/manual-access/invalid/revoke", true, 400},
		{"unauthenticated", "GET", "/api/v1/admin/users", false, 401},
		{"forbidden", "GET", "/api/v1/admin/users", true, 403},
		{"preflight before authentication", "OPTIONS", "/api/v1/admin/users", false, 204},
		{"unknown admin route", "GET", "/api/v1/admin/missing", false, 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			if test.authenticated {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, test.status, response.Code, response.Body.String())
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		})
	}
}
