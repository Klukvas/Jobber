package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/internal/platform/docx"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/pdf"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ExportHandler handles PDF and DOCX export HTTP requests.
type ExportHandler struct {
	service     *service.ResumeBuilderService
	pdfService  *pdf.PDFService
	docxService *docx.DOCXService
	logger      *zap.Logger
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(service *service.ResumeBuilderService, pdfService *pdf.PDFService, logger *zap.Logger) *ExportHandler {
	return &ExportHandler{
		service:    service,
		pdfService: pdfService,
		logger:     logger,
	}
}

// SetDOCXService sets the optional DOCX service for DOCX export.
func (h *ExportHandler) SetDOCXService(svc *docx.DOCXService) {
	h.docxService = svc
}

// RegisterRoutes registers export routes on the given router group.
func (h *ExportHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	group := router.Group("/resume-builder")
	group.POST("/:id/export-pdf", authMiddleware, rateLimiter, h.ExportResumePDF)
	if h.docxService != nil {
		group.POST("/:id/export-docx", authMiddleware, rateLimiter, h.ExportResumeDOCX)
	}
}

// ExportResumePDF exports a resume as a PDF file.
func (h *ExportHandler) ExportResumePDF(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")

	// Get full resume (includes ownership check)
	resume, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Generate PDF — try React frontend rendering first, fallback to Go templates.
	var pdfBytes []byte
	if h.pdfService.HasFrontendPDF() {
		pdfBytes, err = h.pdfService.GenerateResumePDFFromFrontend(c.Request.Context(), resume)
		if err != nil {
			h.logger.Warn("frontend PDF rendering failed, falling back to Go templates",
				zap.String("resume_id", id),
				zap.Error(err),
			)
			pdfBytes = nil
		} else {
			h.logger.Info("PDF generated via React frontend",
				zap.String("resume_id", id),
				zap.Int("size_bytes", len(pdfBytes)),
			)
		}
	} else {
		h.logger.Debug("frontend PDF not configured, using Go templates")
	}
	if pdfBytes == nil {
		pdfBytes, err = h.pdfService.GenerateResumePDF(c.Request.Context(), resume)
		if err != nil {
			httpPlatform.RespondWithError(c, http.StatusInternalServerError, "PDF_GENERATION_FAILED", "Failed to generate PDF")
			return
		}
		h.logger.Info("PDF generated via Go templates",
			zap.String("resume_id", id),
			zap.Int("size_bytes", len(pdfBytes)),
		)
	}

	// Set response headers and send PDF
	setContentDisposition(c, sanitizeFilename(resume.Title)+".pdf")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// ExportResumeDOCX exports a resume as a DOCX file.
func (h *ExportHandler) ExportResumeDOCX(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")

	// Get full resume (includes ownership check)
	resume, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Generate DOCX
	docxBytes, err := h.docxService.GenerateResumeDOCX(resume)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "DOCX_GENERATION_FAILED", "Failed to generate DOCX")
		return
	}

	// Set response headers and send DOCX
	setContentDisposition(c, sanitizeFilename(resume.Title)+".docx")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", docxBytes)
}

// sanitizeFilename removes control characters (including \r, \n) and
// path separators from a filename to prevent HTTP header injection and
// directory traversal.
func sanitizeFilename(name string) string {
	safe := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '/' || r == '\\' || r == ':' || r == '"' || r == '\x00' {
			return '_'
		}
		return r
	}, name)
	if len(safe) > 200 {
		safe = safe[:200]
	}
	if safe == "" {
		safe = "resume"
	}
	return safe
}

// setContentDisposition sets a safe Content-Disposition header with RFC 5987
// encoding for non-ASCII filenames (e.g. Cyrillic, CJK).
func setContentDisposition(c *gin.Context, filename string) {
	encoded := url.PathEscape(filename)
	c.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, encoded))
}

func (h *ExportHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, subModel.ErrLimitReached) {
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the resume builder limit for your current plan.")
		return
	}

	errorCode := model.GetErrorCode(err)
	errorMessage := model.GetErrorMessage(err)

	statusCode := http.StatusInternalServerError
	switch errorCode {
	case model.CodeResumeBuilderNotFound, model.CodeSectionEntryNotFound:
		statusCode = http.StatusNotFound
	case model.CodeNotOwner:
		statusCode = http.StatusForbidden
	case model.CodeInvalidTemplate, model.CodeInvalidSpacing, model.CodeInvalidColor, model.CodeInvalidFont, model.CodeInvalidSectionKey:
		statusCode = http.StatusBadRequest
	}

	httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
}
