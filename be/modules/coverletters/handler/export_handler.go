package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/internal/platform/docx"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/pdf"
	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
)

// ExportHandler handles cover letter PDF and DOCX export HTTP requests.
type ExportHandler struct {
	service     *service.CoverLetterService
	pdfService  *pdf.PDFService
	docxService *docx.DOCXService
}

// NewExportHandler creates a new cover letter ExportHandler.
func NewExportHandler(service *service.CoverLetterService, pdfService *pdf.PDFService) *ExportHandler {
	return &ExportHandler{
		service:    service,
		pdfService: pdfService,
	}
}

// SetDOCXService sets the optional DOCX service for DOCX export.
func (h *ExportHandler) SetDOCXService(svc *docx.DOCXService) {
	h.docxService = svc
}

// RegisterRoutes registers cover letter export routes on the given router group.
func (h *ExportHandler) RegisterRoutes(group *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	g := group.Group("/cover-letters")
	g.POST("/:id/export-pdf", authMiddleware, rateLimiter, h.ExportPDF)
	if h.docxService != nil {
		g.POST("/:id/export-docx", authMiddleware, rateLimiter, h.ExportDOCX)
	}
}

// ExportPDF exports a cover letter as a PDF file.
func (h *ExportHandler) ExportPDF(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")

	// Get cover letter (includes ownership check)
	cl, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Build PDF data
	data := &pdf.CoverLetterPDFData{
		Template:       cl.Template,
		FontFamily:     cl.FontFamily,
		FontSize:       cl.FontSize,
		PrimaryColor:   cl.PrimaryColor,
		RecipientName:  cl.RecipientName,
		RecipientTitle: cl.RecipientTitle,
		CompanyName:    cl.CompanyName,
		CompanyAddress: cl.CompanyAddress,
		Date: time.Now().Format("January 2, 2006"),
		Greeting:   cl.Greeting,
		Paragraphs: cl.Paragraphs,
		Closing:    cl.Closing,
	}

	pdfBytes, err := h.pdfService.GenerateCoverLetterPDF(c.Request.Context(), data)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "PDF_GENERATION_FAILED", "Failed to generate PDF")
		return
	}

	setCLContentDisposition(c, sanitizeCLFilename(cl.Title)+".pdf")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// ExportDOCX exports a cover letter as a DOCX file.
func (h *ExportHandler) ExportDOCX(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")

	cl, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	data := &docx.CoverLetterDOCXData{
		RecipientName:  cl.RecipientName,
		RecipientTitle: cl.RecipientTitle,
		CompanyName:    cl.CompanyName,
		CompanyAddress: cl.CompanyAddress,
		Date:           time.Now().Format("January 2, 2006"),
		Greeting:       cl.Greeting,
		Paragraphs:     cl.Paragraphs,
		Closing:        cl.Closing,
	}

	docxBytes, err := h.docxService.GenerateCoverLetterDOCX(data)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "DOCX_GENERATION_FAILED", "Failed to generate DOCX")
		return
	}

	setCLContentDisposition(c, sanitizeCLFilename(cl.Title)+".docx")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", docxBytes)
}

// sanitizeCLFilename removes control characters and path separators from a filename.
func sanitizeCLFilename(name string) string {
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
		safe = "cover-letter"
	}
	return safe
}

// setCLContentDisposition sets a safe Content-Disposition header with RFC 5987
// encoding for non-ASCII filenames (e.g. Cyrillic, CJK).
func setCLContentDisposition(c *gin.Context, filename string) {
	encoded := url.PathEscape(filename)
	c.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, encoded))
}

func (h *ExportHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subModel.ErrLimitReached):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the cover letter limit for your current plan.")
	case errors.Is(err, model.ErrNotAuthorized):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "NOT_AUTHORIZED", "You don't have access to this cover letter")
	case errors.Is(err, model.ErrCoverLetterNotFound):
		httpPlatform.RespondWithError(c, http.StatusNotFound, "COVER_LETTER_NOT_FOUND", "Cover letter not found")
	default:
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
}
