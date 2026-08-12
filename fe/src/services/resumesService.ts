import { apiClient } from "./api";
import type {
  ResumeDTO,
  CreateResumeRequest,
  UpdateResumeRequest,
  PaginatedResponse,
  GenerateUploadURLRequest,
  GenerateUploadURLResponse,
  DownloadURLResponse,
} from "@/shared/types/api";

export const resumesService = {
  async list(params: {
    limit?: number;
    offset?: number;
    sort_by?: "created_at" | "title" | "is_active";
    sort_dir?: "asc" | "desc";
  }): Promise<PaginatedResponse<ResumeDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit !== undefined)
      searchParams.set("limit", params.limit.toString());
    if (params.offset !== undefined)
      searchParams.set("offset", params.offset.toString());
    if (params.sort_by) searchParams.set("sort_by", params.sort_by);
    if (params.sort_dir) searchParams.set("sort_dir", params.sort_dir);

    return apiClient.get<PaginatedResponse<ResumeDTO>>(
      `resumes?${searchParams.toString()}`,
    );
  },

  async getById(id: string): Promise<ResumeDTO> {
    return apiClient.get<ResumeDTO>(`resumes/${id}`);
  },

  async create(data: CreateResumeRequest): Promise<ResumeDTO> {
    return apiClient.post<ResumeDTO>("resumes", data);
  },

  async update(id: string, data: UpdateResumeRequest): Promise<ResumeDTO> {
    return apiClient.patch<ResumeDTO>(`resumes/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`resumes/${id}`);
  },

  // S3 Upload Methods
  async generateUploadURL(
    request: GenerateUploadURLRequest,
  ): Promise<GenerateUploadURLResponse> {
    return apiClient.post<GenerateUploadURLResponse>(
      "resumes/upload-url",
      request,
    );
  },

  async uploadToS3(uploadUrl: string, file: File): Promise<void> {
    // The presigned URL signs only the host header (X-Amz-SignedHeaders=host),
    // so Content-Type is NOT part of the signature and does not need to match
    // anything. We still send file.type so the object is stored with the right
    // Content-Type.
    // GOTCHA: this cross-origin fetch fails with "Failed to fetch" unless the
    // app origin is allowed BOTH in the bucket CORS rules (be/scripts/setup-cors.go)
    // AND in the CSP connect-src directive (Caddyfile).
    const response = await fetch(uploadUrl, {
      method: "PUT",
      headers: {
        "Content-Type": file.type,
      },
      body: file,
    });

    if (!response.ok) {
      throw new Error("Failed to upload file to S3");
    }
  },

  async generateDownloadURL(id: string): Promise<DownloadURLResponse> {
    return apiClient.get<DownloadURLResponse>(`resumes/${id}/download`);
  },

  // Complete upload flow
  async uploadResume(
    file: File,
    onProgress?: (progress: number) => void,
  ): Promise<ResumeDTO> {
    // Step 1: Generate upload URL
    const uploadData = await this.generateUploadURL({
      filename: file.name,
      content_type: file.type,
    });

    // Step 2: Upload file to S3
    if (onProgress) onProgress(50);
    await this.uploadToS3(uploadData.upload_url, file);
    if (onProgress) onProgress(100);

    // Step 3: Get the created resume
    return this.getById(uploadData.resume_id);
  },
};
