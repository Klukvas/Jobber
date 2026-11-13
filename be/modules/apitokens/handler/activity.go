package handler

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/gin-gonic/gin"
)

func (h *APITokenHandler) Activity(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	limit := model.DefaultActivityLimit
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		h.handleError(c, model.ErrInvalidActivityLimit)
		return
	}
	if values, present := query["limit"]; present {
		if len(values) != 1 {
			h.handleError(c, model.ErrInvalidActivityLimit)
			return
		}
		parsed, err := strconv.Atoi(values[0])
		if err != nil || parsed < 1 || parsed > model.MaxActivityLimit {
			h.handleError(c, model.ErrInvalidActivityLimit)
			return
		}
		limit = parsed
	}
	items, err := h.service.ListActivity(c.Request.Context(), userID, limit)
	if err != nil {
		h.handleError(c, err)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, items)
}
