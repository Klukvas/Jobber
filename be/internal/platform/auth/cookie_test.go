package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCookieConfig(t *testing.T) {
	tests := []struct {
		name             string
		env              string
		expectedSecure   bool
		expectedSameSite http.SameSite
	}{
		{
			name:             "production is secure with SameSiteStrictMode",
			env:              "production",
			expectedSecure:   true,
			expectedSameSite: http.SameSiteStrictMode,
		},
		{
			name:             "development is insecure with SameSiteLaxMode",
			env:              "development",
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "test is same as development",
			env:              "test",
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "empty string treated as non-production",
			env:              "",
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewCookieConfig(tt.env)

			assert.Equal(t, tt.expectedSecure, cfg.Secure)
			assert.Equal(t, tt.expectedSameSite, cfg.SameSite)
			assert.Empty(t, cfg.Domain)
		})
	}
}

// newTestGinContext creates a gin.Context backed by an httptest.ResponseRecorder.
func newTestGinContext(w *httptest.ResponseRecorder) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestSetTokenCookies(t *testing.T) {
	t.Run("sets both cookies with correct attributes", func(t *testing.T) {
		w := httptest.NewRecorder()
		c := newTestGinContext(w)
		cfg := NewCookieConfig("production")

		SetTokenCookies(c, cfg, "access-tok-123", 15*time.Minute, "refresh-tok-456", 7*24*time.Hour)

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 2)

		// Find cookies by name
		var accessCookie, refreshCookie *http.Cookie
		for _, ck := range cookies {
			switch ck.Name {
			case AccessTokenCookie:
				accessCookie = ck
			case RefreshTokenCookie:
				refreshCookie = ck
			}
		}

		require.NotNil(t, accessCookie, "access_token cookie must be set")
		assert.Equal(t, "access-tok-123", accessCookie.Value)
		assert.Equal(t, "/", accessCookie.Path)
		assert.True(t, accessCookie.HttpOnly)
		assert.True(t, accessCookie.Secure)
		assert.Equal(t, int(15*time.Minute.Seconds()), accessCookie.MaxAge)

		require.NotNil(t, refreshCookie, "refresh_token cookie must be set")
		assert.Equal(t, "refresh-tok-456", refreshCookie.Value)
		assert.Equal(t, "/api/v1/auth", refreshCookie.Path)
		assert.True(t, refreshCookie.HttpOnly)
		assert.True(t, refreshCookie.Secure)
		assert.Equal(t, int((7 * 24 * time.Hour).Seconds()), refreshCookie.MaxAge)
	})

	t.Run("development config sets Secure=false", func(t *testing.T) {
		w := httptest.NewRecorder()
		c := newTestGinContext(w)
		cfg := NewCookieConfig("development")

		SetTokenCookies(c, cfg, "at", 10*time.Minute, "rt", 1*time.Hour)

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 2)

		for _, ck := range cookies {
			assert.False(t, ck.Secure, "cookie %s should not be secure in dev", ck.Name)
		}
	})
}

func TestClearTokenCookies(t *testing.T) {
	t.Run("sets cookies with MaxAge=-1", func(t *testing.T) {
		w := httptest.NewRecorder()
		c := newTestGinContext(w)
		cfg := NewCookieConfig("production")

		ClearTokenCookies(c, cfg)

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 2)

		var accessCookie, refreshCookie *http.Cookie
		for _, ck := range cookies {
			switch ck.Name {
			case AccessTokenCookie:
				accessCookie = ck
			case RefreshTokenCookie:
				refreshCookie = ck
			}
		}

		require.NotNil(t, accessCookie)
		assert.Equal(t, "", accessCookie.Value)
		assert.Equal(t, -1, accessCookie.MaxAge)
		assert.Equal(t, "/", accessCookie.Path)
		assert.True(t, accessCookie.HttpOnly)
		assert.True(t, accessCookie.Secure)

		require.NotNil(t, refreshCookie)
		assert.Equal(t, "", refreshCookie.Value)
		assert.Equal(t, -1, refreshCookie.MaxAge)
		assert.Equal(t, "/api/v1/auth", refreshCookie.Path)
		assert.True(t, refreshCookie.HttpOnly)
		assert.True(t, refreshCookie.Secure)
	})

	t.Run("development config clear", func(t *testing.T) {
		w := httptest.NewRecorder()
		c := newTestGinContext(w)
		cfg := NewCookieConfig("development")

		ClearTokenCookies(c, cfg)

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 2)

		for _, ck := range cookies {
			assert.Equal(t, -1, ck.MaxAge)
			assert.Equal(t, "", ck.Value)
			assert.False(t, ck.Secure)
		}
	})
}
