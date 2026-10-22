package interviewprep

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authentication, limiter gin.HandlerFunc) {
	group := router.Group("/jobs/:id/interview-prep", authentication)
	group.GET("", h.history)
	group.GET("/versions/:versionId", h.version)
	group.GET("/sessions/:sessionId", h.session)
	group.GET("/operations/:operationId", h.operation)
	group.POST("/generate", limiter, h.generate)
	group.POST("/operations/:operationId/retry", limiter, h.retry)
	group.PUT("/versions/:versionId/studied", h.studied)
	group.POST("/sessions", h.start)
	group.POST("/sessions/:sessionId/answers", h.answer)
	group.POST("/sessions/:sessionId/action", h.action)
	group.POST("/sessions/:sessionId/answers/:answerId/feedback", limiter, h.feedback)
	group.POST("/sessions/:sessionId/summary", limiter, h.summary)
}
func identity(c *gin.Context) (string, string, bool) {
	owner, ok := auth.GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED")
		return "", "", false
	}
	for _, p := range c.Params {
		if !uuidValid(p.Value) {
			respondError(c, http.StatusBadRequest, "VALIDATION_ERROR")
			return "", "", false
		}
	}
	return owner, c.Param("id"), true
}
func readBody(c *gin.Context, value any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 65536)
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || !utf8.Valid(raw) {
		respondError(c, 400, "VALIDATION_ERROR")
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		respondError(c, 400, "VALIDATION_ERROR")
		return false
	}
	if decoder.Decode(new(any)) != io.EOF {
		respondError(c, 400, "VALIDATION_ERROR")
		return false
	}
	return true
}
func respondError(c *gin.Context, status int, code string) {
	httpPlatform.RespondWithError(c, status, code, "Interview preparation request could not be completed")
}
func result(c *gin.Context, value any, err error) {
	if err == nil {
		httpPlatform.RespondWithData(c, 200, value)
		return
	}
	switch {
	case errors.Is(err, ErrValidation):
		respondError(c, 400, "VALIDATION_ERROR")
	case errors.Is(err, ErrNotFound):
		respondError(c, 404, "NOT_FOUND")
	case errors.Is(err, ErrCapacity):
		respondError(c, http.StatusRequestEntityTooLarge, "ANSWER_CAPACITY_EXCEEDED")
	case errors.Is(err, ErrInputTooLarge):
		respondError(c, http.StatusRequestEntityTooLarge, "SOURCE_CAPACITY_EXCEEDED")
	case errors.Is(err, ErrSourceChanged):
		respondError(c, 409, "SOURCE_CHANGED")
	case errors.Is(err, ErrConflict):
		respondError(c, 409, "CONFLICT")
	case errors.Is(err, ErrUnavailable):
		respondError(c, 503, "AI_UNAVAILABLE")
	case errors.Is(err, ErrRateLimited):
		respondError(c, 429, "RATE_LIMITED")
	case errors.Is(err, subModel.ErrLimitReached):
		respondError(c, 403, "PLAN_LIMIT_REACHED")
	default:
		respondError(c, 500, "INTERNAL_ERROR")
	}
}
func (h *Handler) history(c *gin.Context) {
	owner, job, ok := identity(c)
	if !ok {
		return
	}
	offset := 0
	if c.Query("offset") != "" {
		n, err := strconv.Atoi(c.Query("offset"))
		if err != nil || n < 0 || n > 10000 {
			result(c, nil, ErrValidation)
			return
		}
		offset = n
	}
	v, err := h.service.store.History(c.Request.Context(), owner, job, offset, h.service.Available())
	result(c, v, err)
}
func (h *Handler) version(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	v, e := h.service.store.Version(c.Request.Context(), o, j, c.Param("versionId"))
	result(c, v, e)
}
func (h *Handler) session(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	v, e := h.service.store.Session(c.Request.Context(), o, j, c.Param("sessionId"))
	result(c, v, e)
}
func (h *Handler) generate(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	var req struct {
		ResumeID string `json:"resume_builder_id"`
		Locale   string `json:"locale"`
		Key      string `json:"key"`
	}
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.Generate(c.Request.Context(), o, j, req.ResumeID, req.Locale, req.Key)
	result(c, v, e)
}
func (h *Handler) retry(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	v, e := h.service.Retry(c.Request.Context(), o, j, c.Param("operationId"))
	result(c, v, e)
}
func (h *Handler) studied(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	var req struct {
		QuestionID string `json:"question_id"`
		Studied    *bool  `json:"studied"`
	}
	if !readBody(c, &req) {
		return
	}
	if req.Studied == nil {
		result(c, nil, ErrValidation)
		return
	}
	v, e := h.service.store.Studied(c.Request.Context(), o, j, c.Param("versionId"), req.QuestionID, *req.Studied)
	result(c, v, e)
}
func (h *Handler) start(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	var req struct {
		Version string `json:"version_id"`
		Key     string `json:"key"`
	}
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.store.StartSession(c.Request.Context(), o, j, req.Version, req.Key)
	result(c, v, e)
}
func (h *Handler) answer(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	var req SaveAnswer
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.store.SaveAnswer(c.Request.Context(), o, j, c.Param("sessionId"), req)
	result(c, v, e)
}
func (h *Handler) action(c *gin.Context) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	var req SessionAction
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.store.Action(c.Request.Context(), o, j, c.Param("sessionId"), req)
	result(c, v, e)
}
func (h *Handler) feedback(c *gin.Context) { h.review(c, c.Param("answerId")) }
func (h *Handler) summary(c *gin.Context)  { h.review(c, "") }
func (h *Handler) review(c *gin.Context, answer string) {
	o, j, ok := identity(c)
	if !ok {
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.Review(c.Request.Context(), o, j, c.Param("sessionId"), answer, req.Key)
	result(c, v, e)
}

func (h *Handler) operation(c *gin.Context) {
	owner, job, ok := identity(c)
	if !ok {
		return
	}
	operation, err := scanOperation(h.service.store.pool.QueryRow(c.Request.Context(), operationSelect+` AND id=$3`, owner, job, c.Param("operationId")))
	result(c, operation, err)
}
