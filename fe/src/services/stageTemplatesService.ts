import { apiClient } from "./api";
import type {
  StageTemplateDTO,
  CreateStageTemplateRequest,
  UpdateStageTemplateRequest,
  PaginatedResponse,
} from "@/shared/types/api";

export const stageTemplatesService = {
  async list(params: {
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<StageTemplateDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit !== undefined)
      searchParams.set("limit", params.limit.toString());
    if (params.offset !== undefined)
      searchParams.set("offset", params.offset.toString());

    return apiClient.get<PaginatedResponse<StageTemplateDTO>>(
      `stage-templates?${searchParams.toString()}`,
    );
  },

  async create(data: CreateStageTemplateRequest): Promise<StageTemplateDTO> {
    return apiClient.post<StageTemplateDTO>("stage-templates", data);
  },

  async update(
    id: string,
    data: UpdateStageTemplateRequest,
  ): Promise<StageTemplateDTO> {
    return apiClient.patch<StageTemplateDTO>(`stage-templates/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`stage-templates/${id}`);
  },

  // Persists the full ordered list of stage-template ids (the pipeline order).
  async reorder(stageIds: string[]): Promise<void> {
    return apiClient.post<void>("stage-templates/reorder", {
      stage_ids: stageIds,
    });
  },
};
