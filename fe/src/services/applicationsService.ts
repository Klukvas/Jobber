import { apiClient } from './api';
import type {
  ApplicationDTO,
  CreateApplicationRequest,
  UpdateApplicationRequest,
  PaginatedResponse,
  ApplicationStageDTO,
  AddStageRequest,
  CompleteStageRequest,
} from '@/shared/types/api';

export const applicationsService = {
  async list(params: { 
    limit?: number; 
    offset?: number;
    sort_by?: 'last_activity' | 'status' | 'applied_at';
    sort_dir?: 'asc' | 'desc';
  }): Promise<PaginatedResponse<ApplicationDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit) searchParams.set('limit', params.limit.toString());
    if (params.offset) searchParams.set('offset', params.offset.toString());
    if (params.sort_by) searchParams.set('sort_by', params.sort_by);
    if (params.sort_dir) searchParams.set('sort_dir', params.sort_dir);
    
    return apiClient.get<PaginatedResponse<ApplicationDTO>>(
      `applications?${searchParams.toString()}`
    );
  },

  async getById(id: string): Promise<ApplicationDTO> {
    return apiClient.get<ApplicationDTO>(`applications/${id}`);
  },

  async create(data: CreateApplicationRequest): Promise<ApplicationDTO> {
    return apiClient.post<ApplicationDTO>('applications', data);
  },

  async update(id: string, data: UpdateApplicationRequest): Promise<ApplicationDTO> {
    return apiClient.patch<ApplicationDTO>(`applications/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`applications/${id}`);
  },

  async listStages(id: string): Promise<ApplicationStageDTO[]> {
    return apiClient.get<ApplicationStageDTO[]>(`applications/${id}/stages`);
  },

  async addStage(id: string, data: AddStageRequest): Promise<ApplicationStageDTO> {
    return apiClient.post<ApplicationStageDTO>(`applications/${id}/stages`, data);
  },

  async completeStage(
    id: string,
    stageId: string,
    data?: CompleteStageRequest
  ): Promise<ApplicationStageDTO> {
    return apiClient.patch<ApplicationStageDTO>(
      `applications/${id}/stages/${stageId}/complete`,
      data
    );
  },

  async updateStage(
    id: string,
    stageId: string,
    data: { status?: string; completed_at?: string }
  ): Promise<ApplicationStageDTO> {
    return apiClient.patch<ApplicationStageDTO>(
      `applications/${id}/stages/${stageId}`,
      data
    );
  },

  async deleteStage(id: string, stageId: string): Promise<void> {
    return apiClient.delete<void>(`applications/${id}/stages/${stageId}`);
  },
};
