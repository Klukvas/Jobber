import { apiClient } from './api';
import type { CommentDTO, CreateCommentRequest } from '@/shared/types/api';

export const commentsService = {
  async create(data: CreateCommentRequest): Promise<CommentDTO> {
    return apiClient.post<CommentDTO>('comments', data);
  },

  async listByApplication(applicationId: string): Promise<CommentDTO[]> {
    return apiClient.get<CommentDTO[]>(`applications/${applicationId}/comments`);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`comments/${id}`);
  },
};
