package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/andreypavlenko/jobber/modules/apitokens/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
)

type APITokenHandler struct{ service *service.APITokenService }

func NewAPITokenHandler(service *service.APITokenService) *APITokenHandler {
	return &APITokenHandler{service: service}
}

func (h *APITokenHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api-tokens", authMiddleware)
	group.POST("", h.Create)
	group.GET("", h.List)
	group.GET("/activity", h.Activity)
	group.DELETE("/:id", h.Revoke)
}

func (h *APITokenHandler) Create(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	var req struct {
		Name          string          `json:"name"`
		ExpiresInDays json.RawMessage `json:"expires_in_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}
	days, err := expiryDays(req.ExpiresInDays)
	if err != nil {
		h.handleError(c, err)
		return
	}
	plaintext, token, err := h.service.Create(c.Request.Context(), userID, req.Name, days)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	httpPlatform.RespondWithData(c, http.StatusCreated, struct {
		*model.APITokenDTO
		Token string `json:"token"`
	}{APITokenDTO: token, Token: plaintext})
}

func expiryDays(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 {
		days := 90
		return &days, nil
	}
	var days *int
	if err := json.Unmarshal(raw, &days); err != nil {
		return nil, model.ErrInvalidExpiry
	}
	return days, nil
}

func (h *APITokenHandler) List(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	tokens, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, tokens)
}

func (h *APITokenHandler) Revoke(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	if err := h.service.Revoke(c.Request.Context(), userID, c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *APITokenHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subModel.ErrPaidFeature):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PAID_FEATURE", "AI assistant access is available on paid plans.")
	case errors.Is(err, model.ErrTokenLimit):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "API_TOKEN_LIMIT_REACHED", "You can have at most 10 active API tokens.")
	case errors.Is(err, model.ErrTokenNotFound):
		httpPlatform.RespondWithError(c, http.StatusNotFound, "API_TOKEN_NOT_FOUND", "API token not found")
	case errors.Is(err, model.ErrInvalidName), errors.Is(err, model.ErrInvalidExpiry), errors.Is(err, model.ErrInvalidID), errors.Is(err, model.ErrInvalidActivityLimit):
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	default:
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
}
