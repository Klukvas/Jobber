import { apiClient } from './api';
import type {
  CompanyDTO,
  CreateCompanyRequest,
  UpdateCompanyRequest,
  PaginatedResponse,
} from '@/shared/types/api';

export const companiesService = {
  async list(params: {
    limit?: number;
    offset?: number;
    sort_by?: 'name' | 'last_activity' | 'applications_count';
    sort_dir?: 'asc' | 'desc';
  }): Promise<PaginatedResponse<CompanyDTO>> {
    const searchParams = new URLSearchParams();
    if (params.limit) searchParams.set('limit', params.limit.toString());
    if (params.offset) searchParams.set('offset', params.offset.toString());
    if (params.sort_by) searchParams.set('sort_by', params.sort_by);
    if (params.sort_dir) searchParams.set('sort_dir', params.sort_dir);
    
    return apiClient.get<PaginatedResponse<CompanyDTO>>(
      `companies?${searchParams.toString()}`
    );
  },

  async getById(id: string): Promise<CompanyDTO> {
    return apiClient.get<CompanyDTO>(`companies/${id}`);
  },

  async create(data: CreateCompanyRequest): Promise<CompanyDTO> {
    return apiClient.post<CompanyDTO>('companies', data);
  },

  async update(id: string, data: UpdateCompanyRequest): Promise<CompanyDTO> {
    return apiClient.patch<CompanyDTO>(`companies/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`companies/${id}`);
  },

  async getRelatedCounts(id: string): Promise<{ jobs_count: number; applications_count: number }> {
    return apiClient.get<{ jobs_count: number; applications_count: number }>(
      `companies/${id}/related-counts`
    );
  },
};
