package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// AccessTokenCookie is the cookie name for the access token.
	AccessTokenCookie = "access_token"
	// RefreshTokenCookie is the cookie name for the refresh token.
	RefreshTokenCookie = "refresh_token"
)

// CookieConfig holds settings for auth cookies.
type CookieConfig struct {
	Secure   bool   // true in production (HTTPS only)
	Domain   string // e.g. "jobber-app.com" or "" for same-host
	SameSite http.SameSite
}

// NewCookieConfig creates a CookieConfig based on the server environment.
func NewCookieConfig(env string) CookieConfig {
	if env == "production" {
		return CookieConfig{
			Secure:   true,
			Domain:   "",
			SameSite: http.SameSiteLaxMode,
		}
	}
	return CookieConfig{
		Secure:   false,
		Domain:   "",
		SameSite: http.SameSiteLaxMode,
	}
}

// SetTokenCookies writes httpOnly access and refresh token cookies.
func SetTokenCookies(c *gin.Context, cfg CookieConfig, accessToken string, accessMaxAge time.Duration, refreshToken string, refreshMaxAge time.Duration) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    accessToken,
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   int(accessMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    refreshToken,
		Path:     "/api/v1/auth",
		Domain:   cfg.Domain,
		MaxAge:   int(refreshMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})
}

// ClearTokenCookies removes both auth cookies by setting them to empty with MaxAge -1.
func ClearTokenCookies(c *gin.Context, cfg CookieConfig) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    "",
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    "",
		Path:     "/api/v1/auth",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})
}
