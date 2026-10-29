package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IdentityReader interface {
	GetByID(context.Context, string) (*userModel.User, error)
}
type Store interface {
	List(context.Context, string, int, int) (UserList, error)
	Detail(context.Context, string) (Detail, error)
	Update(context.Context, string, ProfileUpdate) error
	Grant(context.Context, string, string, GrantRequest) error
	Revoke(context.Context, string, string, string, string) error
}
type Handler struct {
	store      Store
	identities IdentityReader
	emails     map[string]bool
	origins    map[string]bool
}

func NewHandler(store Store, identities IdentityReader, emails, origins string) *Handler {
	h := &Handler{store: store, identities: identities, emails: map[string]bool{}, origins: map[string]bool{}}
	for _, email := range strings.Split(emails, ",") {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
			h.emails[email] = true
		}
	}
	for _, origin := range strings.Split(origins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" && origin != "*" {
			h.origins[origin] = true
		}
	}
	return h
}
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authenticate gin.HandlerFunc) {
	group := router.Group("/admin", authenticate, h.authorize)
	group.GET("/access", func(c *gin.Context) { httpPlatform.RespondWithData(c, 200, gin.H{"allowed": true}) })
	group.GET("/users", h.list)
	group.GET("/users/:id", h.detail)
	group.PATCH("/users/:id", h.mutation, h.update)
	group.POST("/users/:id/manual-access", h.mutation, h.grant)
	group.POST("/users/:id/manual-access/:grant/revoke", h.mutation, h.revoke)
}
func fail(c *gin.Context, status int, code, message string) {
	httpPlatform.RespondWithError(c, status, code, message)
	c.Abort()
}
func (h *Handler) authorize(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	id, ok := auth.GetUserID(c)
	if !ok || id == "" {
		fail(c, 401, "UNAUTHORIZED", "Authentication required")
		return
	}
	// Token email claims and browser-persisted roles are never trusted.
	user, err := h.identities.GetByID(c.Request.Context(), id)
	if errors.Is(err, userModel.ErrUserNotFound) {
		fail(c, 403, "FORBIDDEN", "Administrator access required")
		return
	}
	if err != nil {
		fail(c, 503, "UNAVAILABLE", "Could not verify administrator access")
		return
	}
	if user == nil || user.ID != id || !user.EmailVerified || !h.emails[strings.ToLower(strings.TrimSpace(user.Email))] {
		fail(c, 403, "FORBIDDEN", "Administrator access required")
		return
	}
	c.Next()
}
func (h *Handler) mutation(c *gin.Context) {
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" {
		fail(c, 415, "INVALID_CONTENT_TYPE", "Use application/json")
		return
	}
	origin := c.GetHeader("Origin")
	// Cookie-authenticated writes require an explicit trusted browser origin.
	// Header-authenticated clients may omit Origin; a supplied origin is checked.
	if (origin == "" && !(strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ") && strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")) != "")) || (origin != "" && !h.origins[origin]) {
		fail(c, 403, "INVALID_ORIGIN", "Request origin is not allowed")
		return
	}
	c.Next()
}
func bind(c *gin.Context, value any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		fail(c, 400, "INVALID_REQUEST", "Invalid request body")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		fail(c, 400, "INVALID_REQUEST", "Expected one JSON object")
		return false
	}
	return true
}
func userID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	parsed, err := uuid.Parse(id)
	if err != nil {
		fail(c, 400, "INVALID_REQUEST", "Invalid user ID")
		return "", false
	}
	return parsed.String(), true
}
func (h *Handler) respond(c *gin.Context, data any, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		fail(c, 404, "NOT_FOUND", "User or manual grant not found")
	case errors.Is(err, ErrConflict):
		fail(c, 409, "CONFLICT", "An active manual grant exists or this request was changed. Refresh the user details.")
	case err != nil:
		fail(c, 500, "INTERNAL_ERROR", "Could not complete administrator request")
	default:
		httpPlatform.RespondWithData(c, 200, data)
	}
}
func (h *Handler) list(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page > 100000 {
		fail(c, 400, "INVALID_REQUEST", "Page must be between 1 and 100000")
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		fail(c, 400, "INVALID_REQUEST", "Limit must be between 1 and 100")
		return
	}
	search := strings.TrimSpace(c.Query("search"))
	if utf8.RuneCountInString(search) > 200 {
		fail(c, 400, "INVALID_REQUEST", "Search is too long")
		return
	}
	data, err := h.store.List(c.Request.Context(), search, page, limit)
	h.respond(c, data, err)
}
func (h *Handler) detail(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	data, err := h.store.Detail(c.Request.Context(), id)
	h.respond(c, data, err)
}
func validText(value string, min, max int) bool {
	n := utf8.RuneCountInString(value)
	return n >= min && n <= max && !strings.ContainsFunc(value, unicode.IsControl)
}
func (h *Handler) update(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var p ProfileUpdate
	if !bind(c, &p) {
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	if !validText(p.Name, 1, 100) || (p.Locale != "en" && p.Locale != "uk" && p.Locale != "ru") {
		fail(c, 400, "INVALID_REQUEST", "Use a name of 1–100 characters and an allowed locale")
		return
	}
	h.respond(c, gin.H{"saved": true}, h.store.Update(c.Request.Context(), id, p))
}
func (h *Handler) grant(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var p GrantRequest
	if !bind(c, &p) {
		return
	}
	p.Reason = strings.TrimSpace(p.Reason)
	grantID, err := uuid.Parse(p.ID)
	if err != nil || p.Days < 1 || p.Days > 31 || !validText(p.Reason, 3, 500) {
		fail(c, 400, "INVALID_REQUEST", "Use a request UUID, 1–31 days and a reason of 3–500 characters")
		return
	}
	p.ID = grantID.String()
	actor, _ := auth.GetUserID(c)
	h.respond(c, gin.H{"saved": true}, h.store.Grant(c.Request.Context(), id, actor, p))
}
func (h *Handler) revoke(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	grant := c.Param("grant")
	parsedGrant, err := uuid.Parse(grant)
	if err != nil {
		fail(c, 400, "INVALID_REQUEST", "Invalid grant ID")
		return
	}
	var p RevokeRequest
	if !bind(c, &p) {
		return
	}
	p.Reason = strings.TrimSpace(p.Reason)
	if !validText(p.Reason, 3, 500) {
		fail(c, 400, "INVALID_REQUEST", "Use a reason of 3–500 characters")
		return
	}
	actor, _ := auth.GetUserID(c)
	h.respond(c, gin.H{"saved": true}, h.store.Revoke(c.Request.Context(), id, parsedGrant.String(), actor, p.Reason))
}
