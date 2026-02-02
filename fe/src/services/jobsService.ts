import { apiClient } from './api';
import type {
  JobDTO,
  CreateJobRequest,
  UpdateJobRequest,
  PaginatedResponse,
} from '@/shared/types/api';

export interface ListJobsParams {
  limit?: number;
  offset?: number;
  status?: 'active' | 'archived' | 'all';
  sort?: string; // Format: "field:order" (e.g., "created_at:desc", "title:asc")
}

export const jobsService = {
  async list(params: ListJobsParams): Promise<PaginatedResponse<JobDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit) searchParams.set('limit', params.limit.toString());
    if (params.offset) searchParams.set('offset', params.offset.toString());
    if (params.status) searchParams.set('status', params.status);
    if (params.sort) searchParams.set('sort', params.sort);
    
    return apiClient.get<PaginatedResponse<JobDTO>>(
      `jobs?${searchParams.toString()}`
    );
  },

  async getById(id: string): Promise<JobDTO> {
    return apiClient.get<JobDTO>(`jobs/${id}`);
  },

  async create(data: CreateJobRequest): Promise<JobDTO> {
    return apiClient.post<JobDTO>('jobs', data);
  },

  async update(id: string, data: UpdateJobRequest): Promise<JobDTO> {
    return apiClient.patch<JobDTO>(`jobs/${id}`, data);
  },

  async archive(id: string): Promise<JobDTO> {
    return apiClient.patch<JobDTO>(`jobs/${id}`, { status: 'archived' });
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`jobs/${id}`);
  },
};
