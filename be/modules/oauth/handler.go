package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	webAuth "github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	authorizeUserRequestsPerMinute = 30
	authorizeIPRequestsPerMinute   = 60
)

type Handler struct {
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service) *Handler { return &Handler{service: service, logger: zap.NewNop()} }

func (h *Handler) RegisterRoutes(router *gin.Engine, auth gin.HandlerFunc, redis *redis.Client, logger *zap.Logger, origins string) {
	if logger == nil {
		logger = zap.NewNop()
	}
	h.logger = logger
	public := router.Group("/api/oauth", Privacy(), PublicCORS())
	registrationLimit := httpPlatform.RateLimitMiddleware(redis, httpPlatform.RateLimitConfig{MaxRequests: 10, Window: time.Hour, KeyPrefix: "oauth_register"}, logger)
	tokenLimit := httpPlatform.RateLimitMiddleware(redis, httpPlatform.RateLimitConfig{MaxRequests: 60, Window: time.Minute, KeyPrefix: "oauth_token"}, logger)
	public.POST("/register", registrationLimit, h.Register)
	public.POST("/token", tokenLimit, h.Token)
	public.POST("/revoke", tokenLimit, h.Revoke)
	for _, path := range []string{"/register", "/token", "/revoke"} {
		public.OPTIONS(path, func(c *gin.Context) { c.Status(204) })
	}
	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/api/mcp"} {
		router.GET(path, PublicCORS(), h.ProtectedResource)
		router.OPTIONS(path, PublicCORS())
	}
	router.GET("/.well-known/oauth-authorization-server", PublicCORS(), h.Metadata)
	router.OPTIONS("/.well-known/oauth-authorization-server", PublicCORS())
	web := router.Group("/api/v1/oauth", Privacy(), auth)
	ipLimit := httpPlatform.RateLimitMiddleware(redis, httpPlatform.RateLimitConfig{MaxRequests: authorizeIPRequestsPerMinute, Window: time.Minute, KeyPrefix: "oauth_authorize_ip", StandardErrorResponse: true}, logger)
	userLimit := httpPlatform.UserRateLimitMiddleware(redis, httpPlatform.RateLimitConfig{MaxRequests: authorizeUserRequestsPerMinute, Window: time.Minute, KeyPrefix: "oauth_authorize_user", StandardErrorResponse: true}, logger)
	web.GET("/authorize/preview", ipLimit, userLimit, h.Preview)
	web.POST("/authorize", trustedMutation(origins+","+h.service.issuer), ipLimit, userLimit, h.Authorize)
	web.GET("/grants", h.Grants)
	web.DELETE("/grants/:id", trustedMutation(origins+","+h.service.issuer), h.RevokeGrant)
}

func decodeJSON(c *gin.Context, value any) error {
	content, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || content != "application/json" {
		return fmt.Errorf("%w: content type %q", ErrInvalidRequest, truncate(c.GetHeader("Content-Type"), 100))
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidRequest, truncate(err.Error(), 200))
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing data after JSON body", ErrInvalidRequest)
	}
	return nil
}

func (h *Handler) Register(c *gin.Context) {
	var request Client
	if err := decodeJSON(c, &request); err != nil {
		h.logRegistrationReject(c, request, err)
		registrationError(c, ErrInvalidRequest)
		return
	}
	client, err := h.service.Register(c.Request.Context(), request)
	if err != nil {
		h.logRegistrationReject(c, request, err)
		registrationError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(201, client)
}

func (h *Handler) Preview(c *gin.Context) {
	userID, ok := webAuth.MustGetUserID(c)
	if !ok {
		return
	}
	params, err := authorizeQuery(c.Request.URL.RawQuery)
	if err != nil {
		webError(c, err)
		return
	}
	preview, err := h.service.Preview(c.Request.Context(), userID, params)
	if err != nil {
		webError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	if preview.RedirectTo != "" {
		c.JSON(200, gin.H{"redirect_to": preview.RedirectTo})
		return
	}
	c.JSON(200, preview)
}

func authorizeQuery(raw string) (AuthorizeParams, error) {
	values, err := url.ParseQuery(raw)
	if err != nil || len(raw) > 16<<10 {
		return AuthorizeParams{}, ErrInvalidRequest
	}
	for _, entries := range values {
		if len(entries) != 1 {
			return AuthorizeParams{}, ErrInvalidRequest
		}
	}
	return AuthorizeParams{ResponseType: values.Get("response_type"), ClientID: values.Get("client_id"), RedirectURI: values.Get("redirect_uri"), Challenge: values.Get("code_challenge"), ChallengeMethod: values.Get("code_challenge_method"), State: values.Get("state"), Scope: values.Get("scope"), Resource: values.Get("resource")}, nil
}

func (h *Handler) Authorize(c *gin.Context) {
	userID, ok := webAuth.MustGetUserID(c)
	if !ok {
		return
	}
	var params AuthorizeParams
	if err := decodeJSON(c, &params); err != nil {
		webError(c, err)
		return
	}
	redirect, err := h.service.Authorize(c.Request.Context(), userID, params)
	if err != nil {
		webError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"redirect_to": redirect})
}

func (h *Handler) Grants(c *gin.Context) {
	userID, ok := webAuth.MustGetUserID(c)
	if !ok {
		return
	}
	grants, err := h.service.ListGrants(c.Request.Context(), userID)
	if err != nil {
		webError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, grants)
}

func (h *Handler) RevokeGrant(c *gin.Context) {
	userID, ok := webAuth.MustGetUserID(c)
	if !ok {
		return
	}
	if err := h.service.RevokeGrant(c.Request.Context(), userID, c.Param("id")); err != nil {
		webError(c, err)
		return
	}
	c.Status(204)
}

func webError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidClient), errors.Is(err, ErrInvalidRequest):
		httpPlatform.RespondWithError(c, 400, "VALIDATION_ERROR", "Invalid authorization request")
	case errors.Is(err, ErrMetadataBusy):
		c.Header("Retry-After", "5")
		httpPlatform.RespondWithError(c, 503, "OAUTH_METADATA_BUSY", "Client metadata service is busy. Please try again shortly.")
	case errors.Is(err, subModel.ErrPaidFeature):
		httpPlatform.RespondWithError(c, 403, "PAID_FEATURE", "Upgrade to Pro or Enterprise to connect your AI assistant.")
	default:
		httpPlatform.RespondWithError(c, 500, "INTERNAL_ERROR", "Unable to process authorization request")
	}
}

func trustedMutation(origins string) gin.HandlerFunc {
	allowed := strings.Split(origins, ",")
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		trusted := false
		for _, candidate := range allowed {
			if strings.TrimSpace(candidate) != "*" && origin != "" && strings.EqualFold(origin, strings.TrimSpace(candidate)) {
				trusted = true
			}
		}
		// Same policy as admin mutations: cookie writes require a trusted Origin;
		// non-browser JWT clients may omit it. An explicit bad Origin always fails.
		bearer := strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ") && len(strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))) > 0
		if len(c.Request.Header.Values("Origin")) > 1 || !trusted && (origin != "" || !bearer) {
			httpPlatform.RespondWithError(c, 403, "ORIGIN_NOT_ALLOWED", "Trusted origin required")
			c.Abort()
			return
		}
		c.Next()
	}
}
