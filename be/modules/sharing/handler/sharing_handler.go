package handler

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/andreypavlenko/jobber/modules/sharing/ogimage"
	"github.com/andreypavlenko/jobber/modules/sharing/service"
	"github.com/gin-gonic/gin"
)

type SharingHandler struct {
	service     *service.SharingService
	publicBaseURL string
}

func NewSharingHandler(service *service.SharingService, publicBaseURL string) *SharingHandler {
	return &SharingHandler{
		service:     service,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
}

// Create godoc
// @Summary Share current analytics
// @Description Freeze the caller's aggregate stats into a public snapshot and return its share link token
// @Tags sharing
// @Security BearerAuth
// @Produce json
// @Success 201 {object} model.ShareDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 409 {object} httpPlatform.ErrorResponse "Share limit reached"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /shares [post]
func (h *SharingHandler) Create(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	share, err := h.service.Create(c.Request.Context(), userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := string(model.CodeInternalError)
		errorMessage := "Failed to create share"

		if errors.Is(err, model.ErrShareLimitReached) {
			statusCode = http.StatusConflict
			errorCode = string(model.CodeShareLimitReached)
			errorMessage = fmt.Sprintf("You can keep at most %d active share links. Delete one to create a new share.", model.MaxActiveShares)
		}

		httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, share)
}

// List godoc
// @Summary List own shares
// @Description Get all share links created by the caller
// @Tags sharing
// @Security BearerAuth
// @Produce json
// @Success 200 {object} []model.ShareDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /shares [get]
func (h *SharingHandler) List(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	shares, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeInternalError), "Failed to list shares")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, shares)
}

// Delete godoc
// @Summary Revoke a share
// @Description Delete a share link; its public URL stops working immediately
// @Tags sharing
// @Security BearerAuth
// @Produce json
// @Param id path string true "Share ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Share not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /shares/{id} [delete]
func (h *SharingHandler) Delete(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	shareID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, shareID); err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := string(model.CodeInternalError)
		errorMessage := "Failed to delete share"

		if errors.Is(err, model.ErrShareNotFound) {
			statusCode = http.StatusNotFound
			errorCode = string(model.CodeShareNotFound)
			errorMessage = "Share not found"
		}

		httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Share deleted successfully"})
}

// GetPublic godoc
// @Summary Get a shared snapshot (public)
// @Description Get the frozen stats snapshot behind a share token. No authentication.
// @Tags sharing
// @Produce json
// @Param token path string true "Share token"
// @Success 200 {object} model.PublicShareDTO
// @Failure 404 {object} httpPlatform.ErrorResponse "Share not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /public/shares/{token} [get]
func (h *SharingHandler) GetPublic(c *gin.Context) {
	token := c.Param("token")

	share, err := h.service.GetPublicByToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, model.ErrShareNotFound) {
			httpPlatform.RespondWithError(c, http.StatusNotFound, string(model.CodeShareNotFound), "Share not found")
			return
		}
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeInternalError), "Failed to load share")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, share)
}

// PreviewHTML godoc
// @Summary OG preview page for a share (public)
// @Description Minimal HTML with Open Graph meta for social media crawlers. Humans are redirected to the SPA share page.
// @Tags sharing
// @Produce html
// @Param token path string true "Share token"
// @Success 200 {string} string "HTML with OG meta"
// @Failure 404 {string} string "HTML for unknown share"
// @Router /public/shares/{token}/preview-html [get]
func (h *SharingHandler) PreviewHTML(c *gin.Context) {
	token := c.Param("token")

	c.Header("X-Robots-Tag", "noindex")

	share, err := h.service.GetPublicByToken(c.Request.Context(), token)
	if err != nil {
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(notFoundHTML))
		return
	}

	overview := share.Snapshot.Overview
	title := fmt.Sprintf("My job search stats — %d applications, %.0f%% response rate", overview.TotalApplications, overview.ResponseRate)
	description := fmt.Sprintf(
		"%d total applications, %d still in progress, %.0f%% response rate. Tracked with Jobber.",
		overview.TotalApplications, overview.ActiveApplications, overview.ResponseRate,
	)
	shareURL := fmt.Sprintf("%s/s/%s", h.publicBaseURL, url.PathEscape(token))
	imageURL := fmt.Sprintf("%s/api/v1/public/shares/%s/preview-image.png", h.publicBaseURL, url.PathEscape(token))

	page := fmt.Sprintf(previewHTMLTemplate,
		html.EscapeString(title),       // <title>
		html.EscapeString(title),       // og:title
		html.EscapeString(description), // og:description
		html.EscapeString(shareURL),    // og:url
		html.EscapeString(imageURL),    // og:image
		html.EscapeString(imageURL),    // twitter:image
		html.EscapeString(shareURL),    // meta refresh
	)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

// PreviewImage godoc
// @Summary OG preview image for a share (public)
// @Description Dynamically rendered 1200x630 PNG of the share's stats, used as og:image.
// @Tags sharing
// @Produce png
// @Param token path string true "Share token"
// @Success 200 {string} binary "PNG image"
// @Failure 404 {string} string "Unknown share"
// @Router /public/shares/{token}/preview-image.png [get]
func (h *SharingHandler) PreviewImage(c *gin.Context) {
	token := c.Param("token")
	c.Header("X-Robots-Tag", "noindex")

	share, err := h.service.GetPublicByToken(c.Request.Context(), token)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	png, err := ogimage.Render(share.Snapshot)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Snapshots are immutable, so the image can be cached hard.
	c.Header("Cache-Control", "public, max-age=604800, immutable")
	c.Data(http.StatusOK, "image/png", png)
}

func (h *SharingHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, publicRateLimiter gin.HandlerFunc) {
	shares := router.Group("/shares")
	shares.Use(authMiddleware)
	{
		shares.POST("", h.Create)
		shares.GET("", h.List)
		shares.DELETE("/:id", h.Delete)
	}

	// Unauthenticated by design: the token is the only credential. IP rate
	// limiting keeps token scanning impractical.
	public := router.Group("/public/shares")
	public.Use(publicRateLimiter)
	{
		public.GET("/:token", h.GetPublic)
		public.GET("/:token/preview-html", h.PreviewHTML)
		public.GET("/:token/preview-image.png", h.PreviewImage)
	}
}

// previewHTMLTemplate is served to social media crawlers (routed here by
// nginx user-agent matching). The meta refresh forwards humans who open the
// URL directly to the SPA page.
const previewHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>%s</title>
<meta name="robots" content="noindex">
<meta property="og:type" content="website">
<meta property="og:site_name" content="Jobber">
<meta property="og:title" content="%s">
<meta property="og:description" content="%s">
<meta property="og:url" content="%s">
<meta property="og:image" content="%s">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:type" content="image/png">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:image" content="%s">
<meta http-equiv="refresh" content="0;url=%s">
</head>
<body></body>
</html>
`

const notFoundHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Share not found</title>
<meta name="robots" content="noindex">
</head>
<body><p>This share link does not exist or was revoked.</p></body>
</html>
`
