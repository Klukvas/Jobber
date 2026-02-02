import { apiClient } from './api';
import type {
  StageTemplateDTO,
  CreateStageTemplateRequest,
  UpdateStageTemplateRequest,
  PaginatedResponse,
} from '@/shared/types/api';

export const stageTemplatesService = {
  async list(params: { limit?: number; offset?: number }): Promise<PaginatedResponse<StageTemplateDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit) searchParams.set('limit', params.limit.toString());
    if (params.offset) searchParams.set('offset', params.offset.toString());
    
    return apiClient.get<PaginatedResponse<StageTemplateDTO>>(
      `stage-templates?${searchParams.toString()}`
    );
  },

  async create(data: CreateStageTemplateRequest): Promise<StageTemplateDTO> {
    return apiClient.post<StageTemplateDTO>('stage-templates', data);
  },

  async update(id: string, data: UpdateStageTemplateRequest): Promise<StageTemplateDTO> {
    return apiClient.patch<StageTemplateDTO>(`stage-templates/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`stage-templates/${id}`);
  },
};
