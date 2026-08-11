import { apiClient } from "./api";
import type {
  TagDTO,
  CreateTagRequest,
  AttachTagRequest,
  TagEntityType,
} from "@/shared/types/api";

export const tagsService = {
  async list(): Promise<TagDTO[]> {
    return apiClient.get<TagDTO[]>("tags");
  },

  async create(data: CreateTagRequest): Promise<TagDTO> {
    return apiClient.post<TagDTO>("tags", data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`tags/${id}`);
  },

  async attach(tagId: string, data: AttachTagRequest): Promise<void> {
    return apiClient.post<void>(`tags/${tagId}/relations`, data);
  },

  async detach(
    tagId: string,
    entityType: TagEntityType,
    entityId: string,
  ): Promise<void> {
    const params = new URLSearchParams({
      entity_type: entityType,
      entity_id: entityId,
    });
    return apiClient.delete<void>(`tags/${tagId}/relations?${params.toString()}`);
  },

  async listByJob(jobId: string): Promise<TagDTO[]> {
    return apiClient.get<TagDTO[]>(`jobs/${jobId}/tags`);
  },

  async listByCompany(companyId: string): Promise<TagDTO[]> {
    return apiClient.get<TagDTO[]>(`companies/${companyId}/tags`);
  },
};
