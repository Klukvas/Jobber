package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/pdf"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
)

const maxTextImportSize = 100 * 1024 // 100KB

// ImportHandler handles resume import HTTP requests.
type ImportHandler struct {
	importService *service.ImportService
}

// NewImportHandler creates a new ImportHandler.
func NewImportHandler(importService *service.ImportService) *ImportHandler {
	return &ImportHandler{importService: importService}
}

// RegisterRoutes registers import routes on the given router group.
func (h *ImportHandler) RegisterRoutes(group *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	g := group.Group("/resume-builder/import")
	g.POST("/text", authMiddleware, rateLimiter, h.ImportFromText)
	g.POST("/pdf", authMiddleware, rateLimiter, h.ImportFromPDF)
}

type importTextRequest struct {
	Text  string `json:"text" binding:"required"`
	Title string `json:"title"`
}

// ImportFromText imports a resume from pasted text.
func (h *ImportHandler) ImportFromText(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req importTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_REQUEST", "Text is required")
		return
	}

	text := strings.TrimSpace(req.Text)
	if len(text) > maxTextImportSize {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "TEXT_TOO_LARGE", "Text exceeds 100KB limit")
		return
	}
	if text == "" {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "EMPTY_TEXT", "Text cannot be empty")
		return
	}

	result, err := h.importService.ImportFromText(c.Request.Context(), userID, text, req.Title)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

// ImportFromPDF imports a resume from a PDF file upload.
func (h *ImportHandler) ImportFromPDF(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "NO_FILE", "PDF file is required")
		return
	}
	defer file.Close()

	if header.Size > pdf.MaxPDFSize {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "FILE_TOO_LARGE", "PDF file exceeds 5MB limit")
		return
	}

	pdfBytes, err := io.ReadAll(io.LimitReader(file, int64(pdf.MaxPDFSize)+1))
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "READ_FAILED", "Failed to read PDF file")
		return
	}

	if len(pdfBytes) > pdf.MaxPDFSize {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "FILE_TOO_LARGE", "PDF file exceeds 5MB limit")
		return
	}

	title := c.PostForm("title")

	result, err := h.importService.ImportFromPDF(c.Request.Context(), userID, pdfBytes, title)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ImportHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, subModel.ErrLimitReached) {
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the resume builder limit for your current plan.")
		return
	}

	errorCode := model.GetErrorCode(err)
	errorMessage := model.GetErrorMessage(err)

	statusCode := http.StatusInternalServerError
	switch errorCode {
	case model.CodeResumeBuilderNotFound:
		statusCode = http.StatusNotFound
	case model.CodeNotOwner:
		statusCode = http.StatusForbidden
	}

	httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
}
