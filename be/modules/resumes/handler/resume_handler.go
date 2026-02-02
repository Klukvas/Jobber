package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/service"
	"github.com/gin-gonic/gin"
)

type ResumeHandler struct {
	service *service.ResumeService
}

func NewResumeHandler(service *service.ResumeService) *ResumeHandler {
	return &ResumeHandler{service: service}
}

// Create godoc
// @Summary Create a new resume
// @Description Create a new resume version for the authenticated user
// @Tags resumes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateResumeRequest true "Resume details"
// @Success 201 {object} model.ResumeDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes [post]
func (h *ResumeHandler) Create(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	var req model.CreateResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	resume, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, resume)
}

// Get godoc
// @Summary Get a resume
// @Description Get details of a specific resume by ID
// @Tags resumes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Resume ID"
// @Success 200 {object} model.ResumeDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Resume not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes/{id} [get]
func (h *ResumeHandler) Get(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	resumeID := c.Param("id")

	resume, err := h.service.GetByID(c.Request.Context(), userID, resumeID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeResumeNotFound {
			statusCode = http.StatusNotFound
		}
		httpPlatform.RespondWithError(c, statusCode, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, resume)
}

// List godoc
// @Summary List resumes
// @Description Get a paginated list of resume versions for the authenticated user
// @Tags resumes
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items per page (default: 20, max: 100)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Success 200 {object} httpPlatform.PaginatedResponse{items=[]model.ResumeDTO}
// @Failure 400 {object} httpPlatform.ErrorResponse "Invalid pagination parameters"
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes [get]
func (h *ResumeHandler) List(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	
	// Parse pagination parameters
	pagination, err := httpPlatform.ParsePaginationParams(c)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_PAGINATION_PARAMS", "Invalid pagination parameters")
		return
	}

	// Parse sorting parameters
	sortBy := c.Query("sort_by")
	sortDir := c.Query("sort_dir")

	resumes, total, err := h.service.List(c.Request.Context(), userID, pagination.Limit, pagination.Offset, sortBy, sortDir)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list resumes")
		return
	}
	httpPlatform.RespondWithPagination(c, http.StatusOK, resumes, pagination.Limit, pagination.Offset, total)
}

// Update godoc
// @Summary Update a resume
// @Description Update details of a specific resume
// @Tags resumes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Resume ID"
// @Param request body model.UpdateResumeRequest true "Updated resume details"
// @Success 200 {object} model.ResumeDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Resume not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes/{id} [patch]
func (h *ResumeHandler) Update(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	resumeID := c.Param("id")
	var req model.UpdateResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	resume, err := h.service.Update(c.Request.Context(), userID, resumeID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeResumeNotFound {
			statusCode = http.StatusNotFound
		}
		httpPlatform.RespondWithError(c, statusCode, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, resume)
}

// Delete godoc
// @Summary Delete a resume
// @Description Delete a specific resume by ID
// @Tags resumes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Resume ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Resume not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes/{id} [delete]
func (h *ResumeHandler) Delete(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	resumeID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, resumeID); err != nil {
		statusCode := http.StatusInternalServerError
		errCode := model.GetErrorCode(err)
		
		// Map error codes to appropriate HTTP status codes
		switch errCode {
		case model.CodeResumeNotFound:
			statusCode = http.StatusNotFound
		case model.CodeResumeInUse:
			statusCode = http.StatusBadRequest
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errCode), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Resume deleted successfully"})
}

// GenerateUploadURL godoc
// @Summary Generate presigned upload URL
// @Description Generate a presigned URL for uploading a resume file to S3
// @Tags resumes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.GenerateUploadURLRequest true "Upload request"
// @Success 200 {object} model.GenerateUploadURLResponse
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes/upload-url [post]
func (h *ResumeHandler) GenerateUploadURL(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	var req model.GenerateUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	response, err := h.service.GenerateUploadURL(c.Request.Context(), userID, &req)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "UPLOAD_URL_GENERATION_FAILED", err.Error())
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, response)
}

// DownloadResume godoc
// @Summary Download resume file
// @Description Generate a presigned URL for downloading a resume file from S3
// @Tags resumes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Resume ID"
// @Success 200 {object} model.DownloadURLResponse
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Resume not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes/{id}/download [get]
func (h *ResumeHandler) DownloadResume(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	resumeID := c.Param("id")

	response, err := h.service.GenerateDownloadURL(c.Request.Context(), userID, resumeID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeResumeNotFound {
			statusCode = http.StatusNotFound
		}
		httpPlatform.RespondWithError(c, statusCode, "DOWNLOAD_URL_GENERATION_FAILED", err.Error())
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, response)
}

func (h *ResumeHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	resumes := router.Group("/resumes")
	resumes.Use(authMiddleware)
	{
		resumes.POST("", h.Create)
		resumes.POST("/upload-url", h.GenerateUploadURL)
		resumes.GET("", h.List)
		resumes.GET("/:id", h.Get)
		resumes.GET("/:id/download", h.DownloadResume)
		resumes.PATCH("/:id", h.Update)
		resumes.DELETE("/:id", h.Delete)
	}
}
