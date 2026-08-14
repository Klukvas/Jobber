import { apiClient } from "./api";
import type {
  MoveTarget,
  JobDTO,
  JobStatus,
  CreateJobRequest,
  UpdateJobRequest,
  PaginatedResponse,
  JobStageDTO,
  AddStageRequest,
  UpdateStageRequest,
} from "@/shared/types/api";

export interface ListJobsParams {
  limit?: number;
  offset?: number;
  /**
   * Omitted / "active" => everything except archived (legacy-compatible),
   * "all" => no filter, or one of the six job statuses.
   */
  status?: JobStatus | "active" | "all";
  sort?: string; // Format: "field:dir" (e.g., "last_activity:desc", "title:asc")
  /** Case-insensitive search matched against job title and company name. */
  search?: string;
}

export const jobsService = {
  async list(params: ListJobsParams): Promise<PaginatedResponse<JobDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit !== undefined)
      searchParams.set("limit", params.limit.toString());
    if (params.offset !== undefined)
      searchParams.set("offset", params.offset.toString());
    if (params.status) searchParams.set("status", params.status);
    if (params.sort) searchParams.set("sort", params.sort);
    if (params.search) searchParams.set("search", params.search);

    return apiClient.get<PaginatedResponse<JobDTO>>(
      `jobs?${searchParams.toString()}`,
    );
  },

  async getById(id: string): Promise<JobDTO> {
    return apiClient.get<JobDTO>(`jobs/${id}`);
  },

  async create(data: CreateJobRequest): Promise<JobDTO> {
    return apiClient.post<JobDTO>("jobs", data);
  },

  async update(id: string, data: UpdateJobRequest): Promise<JobDTO> {
    return apiClient.patch<JobDTO>(`jobs/${id}`, data);
  },

  async archive(id: string): Promise<JobDTO> {
    return apiClient.patch<JobDTO>(`jobs/${id}`, { status: "archived" });
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`jobs/${id}`);
  },

  // Atomic unified-board move: sets stage/status consistently on the backend
  async move(id: string, target: MoveTarget): Promise<JobDTO> {
    return apiClient.post<JobDTO>(`jobs/${id}/move`, { target });
  },

  async toggleFavorite(id: string): Promise<{ is_favorite: boolean }> {
    return apiClient.post<{ is_favorite: boolean }>(`jobs/${id}/favorite`);
  },

  async listStages(id: string): Promise<JobStageDTO[]> {
    return apiClient.get<JobStageDTO[]>(`jobs/${id}/stages`);
  },

  async addStage(id: string, data: AddStageRequest): Promise<JobStageDTO> {
    return apiClient.post<JobStageDTO>(`jobs/${id}/stages`, data);
  },

  async updateStage(
    id: string,
    stageId: string,
    data: UpdateStageRequest,
  ): Promise<JobStageDTO> {
    return apiClient.patch<JobStageDTO>(`jobs/${id}/stages/${stageId}`, data);
  },

  async deleteStage(id: string, stageId: string): Promise<void> {
    return apiClient.delete<void>(`jobs/${id}/stages/${stageId}`);
  },
};
