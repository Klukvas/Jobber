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
	"github.com/andreypavlenko/jobber/internal/platform/ids"
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

// pathIDs holds a request's path parameters in canonical uuid form, keyed by
// parameter name. Handlers read ids from here, never from c.Param: uuid.Parse
// accepts spellings such as `urn:uuid:…` that Postgres rejects.
type pathIDs map[string]string

// identity returns the caller and the request's path ids, answering the request
// itself when the caller is unauthenticated or any path parameter is not a uuid.
// The job id is params["id"].
func identity(c *gin.Context) (string, pathIDs, bool) {
	owner, ok := auth.GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED")
		return "", nil, false
	}
	params := make(pathIDs, len(c.Params))
	for _, p := range c.Params {
		canonical, valid := ids.CanonicalUUID(p.Value)
		if !valid {
			respondError(c, http.StatusBadRequest, "VALIDATION_ERROR")
			return "", nil, false
		}
		params[p.Key] = canonical
	}
	return owner, params, true
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
	if httpPlatform.RespondAIError(c, err) {
		return
	}
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
	owner, params, ok := identity(c)
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
	v, err := h.service.store.History(c.Request.Context(), owner, params["id"], offset, h.service.Available())
	result(c, v, err)
}
func (h *Handler) version(c *gin.Context) {
	o, p, ok := identity(c)
	if !ok {
		return
	}
	v, e := h.service.store.Version(c.Request.Context(), o, p["id"], p["versionId"])
	result(c, v, e)
}
func (h *Handler) session(c *gin.Context) {
	o, p, ok := identity(c)
	if !ok {
		return
	}
	v, e := h.service.store.Session(c.Request.Context(), o, p["id"], p["sessionId"])
	result(c, v, e)
}
func (h *Handler) generate(c *gin.Context) {
	o, p, ok := identity(c)
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
	v, e := h.service.Generate(c.Request.Context(), o, p["id"], req.ResumeID, req.Locale, req.Key)
	result(c, v, e)
}
func (h *Handler) retry(c *gin.Context) {
	o, p, ok := identity(c)
	if !ok {
		return
	}
	v, e := h.service.Retry(c.Request.Context(), o, p["id"], p["operationId"])
	result(c, v, e)
}
func (h *Handler) studied(c *gin.Context) {
	o, p, ok := identity(c)
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
	v, e := h.service.store.Studied(c.Request.Context(), o, p["id"], p["versionId"], req.QuestionID, *req.Studied)
	result(c, v, e)
}
func (h *Handler) start(c *gin.Context) {
	o, p, ok := identity(c)
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
	v, e := h.service.store.StartSession(c.Request.Context(), o, p["id"], req.Version, req.Key)
	result(c, v, e)
}
func (h *Handler) answer(c *gin.Context) {
	o, p, ok := identity(c)
	if !ok {
		return
	}
	var req SaveAnswer
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.store.SaveAnswer(c.Request.Context(), o, p["id"], p["sessionId"], req)
	result(c, v, e)
}
func (h *Handler) action(c *gin.Context) {
	o, p, ok := identity(c)
	if !ok {
		return
	}
	var req SessionAction
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.store.Action(c.Request.Context(), o, p["id"], p["sessionId"], req)
	result(c, v, e)
}
func (h *Handler) feedback(c *gin.Context) { h.review(c, "answerId") }
func (h *Handler) summary(c *gin.Context)  { h.review(c, "") }

// review requests feedback on the answer named by answerParam, or a session
// summary when answerParam is empty.
func (h *Handler) review(c *gin.Context, answerParam string) {
	o, p, ok := identity(c)
	if !ok {
		return
	}
	answer, found := p[answerParam]
	if answerParam != "" && !found {
		// A route without the parameter would silently turn feedback into a
		// differently charged summary; refuse instead.
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR")
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if !readBody(c, &req) {
		return
	}
	v, e := h.service.Review(c.Request.Context(), o, p["id"], p["sessionId"], answer, req.Key)
	result(c, v, e)
}

func (h *Handler) operation(c *gin.Context) {
	owner, params, ok := identity(c)
	if !ok {
		return
	}
	operation, err := scanOperation(h.service.store.pool.QueryRow(c.Request.Context(), operationSelect+` AND id=$3`, owner, params["id"], params["operationId"]))
	result(c, operation, err)
}
