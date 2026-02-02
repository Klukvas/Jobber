package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/companies/service"
	"github.com/gin-gonic/gin"
)

// CompanyHandler handles company HTTP requests
type CompanyHandler struct {
	service *service.CompanyService
}

// NewCompanyHandler creates a new company handler
func NewCompanyHandler(service *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{service: service}
}

// Create godoc
// @Summary Create a new company
// @Description Create a new company for the authenticated user
// @Tags companies
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateCompanyRequest true "Company details"
// @Success 201 {object} model.CompanyDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /companies [post]
func (h *CompanyHandler) Create(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	company, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeCompanyNameRequired {
			statusCode = http.StatusBadRequest
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, company)
}

// Get godoc
// @Summary Get a company
// @Description Get details of a specific company by ID
// @Tags companies
// @Security BearerAuth
// @Produce json
// @Param id path string true "Company ID"
// @Success 200 {object} model.CompanyDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Company not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /companies/{id} [get]
func (h *CompanyHandler) Get(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	companyID := c.Param("id")

	company, err := h.service.GetByID(c.Request.Context(), userID, companyID)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeCompanyNotFound {
			statusCode = http.StatusNotFound
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, company)
}

// List godoc
// @Summary List companies
// @Description Get a paginated list of companies for the authenticated user with enriched fields
// @Tags companies
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items per page (default: 20, max: 100)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Param sort_by query string false "Sort field: name, last_activity, applications_count (default: name)"
// @Param sort_dir query string false "Sort direction: asc, desc (default: asc)"
// @Success 200 {object} httpPlatform.PaginatedResponse{items=[]model.CompanyDTO}
// @Failure 400 {object} httpPlatform.ErrorResponse "Invalid pagination parameters"
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /companies [get]
func (h *CompanyHandler) List(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	// Parse pagination parameters
	pagination, err := httpPlatform.ParsePaginationParams(c)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_PAGINATION_PARAMS", "Invalid pagination parameters")
		return
	}

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "name")
	sortDir := c.DefaultQuery("sort_dir", "asc")

	// Validate sort_by
	validSortFields := map[string]bool{
		"name":               true,
		"last_activity":      true,
		"applications_count": true,
	}
	if !validSortFields[sortBy] {
		sortBy = "name"
	}

	opts := &ports.ListOptions{
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		SortBy:  sortBy,
		SortDir: sortDir,
	}

	companies, total, err := h.service.List(c.Request.Context(), userID, opts)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list companies")
		return
	}

	httpPlatform.RespondWithPagination(c, http.StatusOK, companies, pagination.Limit, pagination.Offset, total)
}

// Update godoc
// @Summary Update a company
// @Description Update details of a specific company
// @Tags companies
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Company ID"
// @Param request body model.UpdateCompanyRequest true "Updated company details"
// @Success 200 {object} model.CompanyDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Company not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /companies/{id} [patch]
func (h *CompanyHandler) Update(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	companyID := c.Param("id")

	var req model.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	company, err := h.service.Update(c.Request.Context(), userID, companyID, &req)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeCompanyNotFound {
			statusCode = http.StatusNotFound
		} else if errorCode == model.CodeCompanyNameRequired {
			statusCode = http.StatusBadRequest
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, company)
}

// Delete godoc
// @Summary Delete a company
// @Description Delete a specific company by ID
// @Tags companies
// @Security BearerAuth
// @Produce json
// @Param id path string true "Company ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Company not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /companies/{id} [delete]
func (h *CompanyHandler) Delete(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	companyID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, companyID); err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeCompanyNotFound {
			statusCode = http.StatusNotFound
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Company deleted successfully"})
}

// GetRelatedCounts godoc
// @Summary Get related jobs and applications count
// @Description Get counts of jobs and applications related to a company for delete warning
// @Tags companies
// @Security BearerAuth
// @Produce json
// @Param id path string true "Company ID"
// @Success 200 {object} map[string]int "Returns jobs_count and applications_count"
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Company not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /companies/{id}/related-counts [get]
func (h *CompanyHandler) GetRelatedCounts(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	companyID := c.Param("id")

	jobsCount, appsCount, err := h.service.GetRelatedJobsAndApplicationsCount(c.Request.Context(), userID, companyID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get related counts")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{
		"jobs_count":         jobsCount,
		"applications_count": appsCount,
	})
}

// RegisterRoutes registers company routes
func (h *CompanyHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	companies := router.Group("/companies")
	companies.Use(authMiddleware)
	{
		companies.POST("", h.Create)
		companies.GET("", h.List)
		companies.GET("/:id", h.Get)
		companies.GET("/:id/related-counts", h.GetRelatedCounts)
		companies.PATCH("/:id", h.Update)
		companies.DELETE("/:id", h.Delete)
	}
}
