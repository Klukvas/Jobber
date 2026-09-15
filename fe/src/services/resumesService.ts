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

  /**
   * Asks the server to verify the object that actually landed in storage and
   * activate the resume. This is where a spoofed .pdf is caught — the browser
   * uploads straight to storage, so nothing before this point is trustworthy.
   */
  async finalizeUpload(id: string, title?: string): Promise<ResumeDTO> {
    return apiClient.post<ResumeDTO>(
      `resumes/${id}/finalize`,
      title ? { title } : {},
    );
  },

  // Complete upload flow
  async uploadResume(
    file: File,
    onProgress?: (progress: number) => void,
    title?: string,
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

    // Step 3: Server-side verification + activation. A rejected upload is
    // deleted server-side, so no half-created resume is left behind.
    return this.finalizeUpload(uploadData.resume_id, title);
  },
};

/** First bytes of every PDF file. */
const PDF_MAGIC = [0x25, 0x50, 0x44, 0x46, 0x2d]; // %PDF-

/**
 * Reads the file's first bytes to check it really is a PDF.
 *
 * This is a UX affordance, not a security control: it runs in the browser and
 * can be bypassed. The upload is only trusted once the server has verified the
 * stored object in `finalizeUpload`.
 */
export async function looksLikePdf(file: File): Promise<boolean> {
  const header = new Uint8Array(
    await file.slice(0, PDF_MAGIC.length).arrayBuffer(),
  );
  return PDF_MAGIC.every((byte, index) => header[index] === byte);
}
