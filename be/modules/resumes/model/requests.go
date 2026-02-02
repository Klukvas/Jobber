package model

type CreateResumeRequest struct {
	Title    string  `json:"title" binding:"required,min=1,max=255"`
	FileURL  *string `json:"file_url,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type UpdateResumeRequest struct {
	Title    *string `json:"title,omitempty"`
	FileURL  *string `json:"file_url,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// GenerateUploadURLRequest represents request for generating presigned upload URL
type GenerateUploadURLRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required,eq=application/pdf"`
}

// GenerateUploadURLResponse represents response with presigned upload URL
type GenerateUploadURLResponse struct {
	ResumeID  string `json:"resume_id"`
	UploadURL string `json:"upload_url"`
	ExpiresIn int    `json:"expires_in"`
}

// DownloadURLResponse represents response with presigned download URL
type DownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
	ExpiresIn   int    `json:"expires_in"`
}
