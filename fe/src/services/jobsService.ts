import { apiClient, type RequestOptions } from "./api";
import type {
  MoveJobRequest,
  JobDTO,
  CreateJobRequest,
  UpdateJobRequest,
  PaginatedResponse,
  JobStageDTO,
  AddStageRequest,
  UpdateStageRequest,
} from "@/shared/types/api";

/** Archived filter passed as the list `status` query param. */
export type ArchivedFilter = "" | "active" | "archived" | "all";

export interface ListJobsParams {
  limit?: number;
  offset?: number;
  /**
   * Archived filter: "" / "active" => not archived, "archived" => only
   * archived, "all" => both.
   */
  status?: ArchivedFilter;
  sort?: string; // Format: "field:dir" (e.g., "last_activity:desc", "title:asc")
  /** Case-insensitive search matched against job title and company name. */
  search?: string;
  /** Restrict the list to cards linked to one company (UUID). */
  company_id?: string;
}

export const jobsService = {
  /**
   * The list behind the search box and the board.
   *
   * `options.signal` is React Query's: a keystroke changes the key and the
   * request for the previous one is abandoned, so it may as well be cancelled
   * on the wire too. Superseded searches otherwise ran to completion and, on a
   * slow connection, queued behind the one whose answer is actually wanted.
   */
  async list(
    params: ListJobsParams,
    options?: RequestOptions,
  ): Promise<PaginatedResponse<JobDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit !== undefined)
      searchParams.set("limit", params.limit.toString());
    if (params.offset !== undefined)
      searchParams.set("offset", params.offset.toString());
    if (params.status) searchParams.set("status", params.status);
    if (params.sort) searchParams.set("sort", params.sort);
    if (params.search) searchParams.set("search", params.search);
    if (params.company_id) searchParams.set("company_id", params.company_id);

    return apiClient.get<PaginatedResponse<JobDTO>>(
      `jobs?${searchParams.toString()}`,
      options,
    );
  },

  async getById(id: string, options?: RequestOptions): Promise<JobDTO> {
    return apiClient.get<JobDTO>(`jobs/${id}`, options);
  },

  async create(data: CreateJobRequest): Promise<JobDTO> {
    return apiClient.post<JobDTO>("jobs", data);
  },

  async update(id: string, data: UpdateJobRequest): Promise<JobDTO> {
    return apiClient.patch<JobDTO>(`jobs/${id}`, data);
  },

  async archive(id: string): Promise<JobDTO> {
    return apiClient.patch<JobDTO>(`jobs/${id}`, { is_archived: true });
  },

  async unarchive(id: string): Promise<JobDTO> {
    return apiClient.patch<JobDTO>(`jobs/${id}`, { is_archived: false });
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`jobs/${id}`);
  },

  // Moves the card to a pipeline column — the single write path for its state.
  async move(id: string, data: MoveJobRequest): Promise<JobDTO> {
    return apiClient.post<JobDTO>(`jobs/${id}/move`, data);
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
