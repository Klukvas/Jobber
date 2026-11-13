package mcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	patService "github.com/andreypavlenko/jobber/modules/apitokens/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	maxBodyBytes           = 1 << 20
	mcpRequestsPerMinute   = 60
	mcpIPRequestsPerMinute = 120
)

type httpConfig struct {
	allowedOrigins string
	ipLimiter      gin.HandlerFunc
	userLimiter    gin.HandlerFunc
}

type TokenAuthenticator interface {
	AuthenticateWithInfo(context.Context, string) (*patModel.AuthenticationInfo, error)
}

type verifiedPATKey struct{}

// RegisterRoutes mounts the PAT-only surface separately from JWT API routes.
func RegisterRoutes(router *gin.Engine, server *sdk.Server, tokens TokenAuthenticator, paid PaidPlanChecker, redisClient *redis.Client, logger *zap.Logger, allowedOrigins string) {
	if logger == nil {
		logger = zap.NewNop()
	}
	limiter := httpPlatform.UserRateLimitMiddleware(redisClient, httpPlatform.RateLimitConfig{
		MaxRequests: mcpRequestsPerMinute, Window: time.Minute, KeyPrefix: "mcp",
	}, logger)
	ipLimiter := httpPlatform.RateLimitMiddleware(redisClient, httpPlatform.RateLimitConfig{
		MaxRequests: mcpIPRequestsPerMinute, Window: time.Minute, KeyPrefix: "mcp_ip",
	}, logger)
	registerHTTP(router, server, tokens, paid, httpConfig{allowedOrigins: allowedOrigins, ipLimiter: ipLimiter, userLimiter: limiter}, logger)
}

func registerHTTP(router *gin.Engine, server *sdk.Server, tokens TokenAuthenticator, paid PaidPlanChecker, config httpConfig, logger *zap.Logger) {
	transport := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	handler := auth.RequireBearerToken(cachedVerifier, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true, Scopes: []string{"mcp"}})(transport)
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		router.Handle(method, "/api/mcp", redactAuditRequest, bearerChallenge(), requireOrigin(config.allowedOrigins), config.ipLimiter, authenticatePAT(tokens), requirePaid(paid, logger), config.userLimiter, capBody(), gin.WrapH(handler))
	}
}

func authenticatePAT(tokens TokenAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		fields := strings.Fields(c.GetHeader("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || !strings.HasPrefix(fields[1], patService.TokenPrefix) {
			rejectAuth(c)
			return
		}
		info, err := tokens.AuthenticateWithInfo(c.Request.Context(), fields[1])
		if err != nil || info == nil || info.UserID == "" || info.TokenID == "" || (!info.ExpiresAt.IsZero() && !info.ExpiresAt.After(time.Now())) {
			rejectAuth(c)
			return
		}
		c.Set("user_id", info.UserID)
		c.Set("auth_via", "pat")
		c.Set("api_token_id", info.TokenID)
		tokenInfo := &auth.TokenInfo{UserID: info.UserID, Expiration: info.ExpiresAt, Scopes: []string{"mcp"}, Extra: map[string]any{"api_token_id": info.TokenID}}
		ctx := context.WithValue(c.Request.Context(), verifiedPATKey{}, tokenInfo)
		ctx = context.WithValue(ctx, requestIDKey{}, c.GetString("request_id"))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Only the preceding middleware can populate this private context key. The SDK
// verifier reuses that identity, so each HTTP request has exactly one DB lookup.
func cachedVerifier(ctx context.Context, _ string, _ *http.Request) (*auth.TokenInfo, error) {
	info, ok := ctx.Value(verifiedPATKey{}).(*auth.TokenInfo)
	if !ok || info == nil {
		return nil, auth.ErrInvalidToken
	}
	return info, nil
}

func rejectAuth(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
	c.Abort()
}

func requirePaid(paid PaidPlanChecker, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := paid.RequirePaidPlan(c.Request.Context(), c.GetString("user_id"))
		if err == nil {
			c.Next()
			return
		}
		if errors.Is(err, subModel.ErrPaidFeature) {
			httpPlatform.RespondWithError(c, http.StatusForbidden, "PAID_FEATURE", "Upgrade to Pro or Enterprise to connect your AI assistant.")
		} else {
			logger.Error("MCP paid plan check failed", zap.String("request_id", c.GetString("request_id")))
			httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to check your plan. Please try again later.")
		}
		c.Abort()
	}
}

func capBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read at most cap+1 before handing off: SDK decoder failures otherwise turn
		// MaxBytesReader errors into a protocol 400 instead of a clear HTTP 413.
		if c.Request.Body == nil {
			c.Request.Body = http.NoBody
		}
		original := c.Request.Body
		defer original.Close()
		body, err := io.ReadAll(io.LimitReader(original, maxBodyBytes+1))
		if len(body) > maxBodyBytes {
			httpPlatform.RespondWithError(c, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body must be at most 1 MiB.")
			c.Abort()
			return
		}
		if err != nil {
			httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_BODY", "Unable to read request body.")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Next()
	}
}
