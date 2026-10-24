package handler

import (
	"context"
	"net/http"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PDFTailoringService interface {
	Import(context.Context, string, string, *model.PDFImportRequest) (*model.PDFImportResult, error)
	ConfirmRecognition(context.Context, string, string, string, string) error
	SourceReview(context.Context, string, string, string) (string, bool, error)
}
type PDFTailoringHandler struct{ service PDFTailoringService }

func NewPDFTailoringHandler(service PDFTailoringService) *PDFTailoringHandler {
	return &PDFTailoringHandler{service: service}
}
func (h *PDFTailoringHandler) RegisterRoutes(router *gin.RouterGroup, auth, limit gin.HandlerFunc) {
	router.GET("/jobs/:id/resume-tailoring/source/:sourceId", auth, h.SourceReview)
	router.POST("/jobs/:id/resume-tailoring/import", auth, limit, h.Import)
	router.POST("/jobs/:id/resume-tailoring/recognition", auth, limit, h.ConfirmRecognition)
}
func (h *PDFTailoringHandler) Import(c *gin.Context) {
	user, ok := tailoringUser(c)
	if !ok {
		return
	}
	var req model.PDFImportRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
	if err := c.ShouldBindJSON(&req); err != nil {
		tailoringError(c, model.ErrTailoringValidation)
		return
	}
	result, err := h.service.Import(c.Request.Context(), user, c.Param("id"), &req)
	if err != nil {
		tailoringError(c, err)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, result)
}
func (h *PDFTailoringHandler) ConfirmRecognition(c *gin.Context) {
	user, ok := tailoringUser(c)
	if !ok {
		return
	}
	var req struct {
		ImportID string `json:"import_id" binding:"required,uuid"`
		SourceID string `json:"resume_builder_id" binding:"required,uuid"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&req); err != nil {
		tailoringError(c, model.ErrTailoringValidation)
		return
	}
	if err := h.service.ConfirmRecognition(c.Request.Context(), user, c.Param("id"), req.ImportID, req.SourceID); err != nil {
		tailoringError(c, err)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"confirmed": true})
}

func (h *PDFTailoringHandler) SourceReview(c *gin.Context) {
	user, ok := tailoringUser(c)
	if !ok {
		return
	}
	if _, err := uuid.Parse(c.Param("sourceId")); err != nil {
		tailoringError(c, model.ErrTailoringValidation)
		return
	}
	importID, required, err := h.service.SourceReview(c.Request.Context(), user, c.Param("id"), c.Param("sourceId"))
	if err != nil {
		tailoringError(c, err)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"import_id": importID, "requires_review": required})
}
