import { apiClient } from "./api";
import type {
  CoverLetterDTO,
  CreateCoverLetterRequest,
  UpdateCoverLetterRequest,
} from "@/shared/types/cover-letter";

const BASE = "cover-letters";

export const coverLetterService = {
  async list(): Promise<CoverLetterDTO[]> {
    return apiClient.get<CoverLetterDTO[]>(BASE);
  },

  async getById(id: string): Promise<CoverLetterDTO> {
    return apiClient.get<CoverLetterDTO>(`${BASE}/${id}`);
  },

  async create(data: CreateCoverLetterRequest): Promise<CoverLetterDTO> {
    return apiClient.post<CoverLetterDTO>(BASE, data);
  },

  async update(
    id: string,
    data: UpdateCoverLetterRequest,
  ): Promise<CoverLetterDTO> {
    return apiClient.patch<CoverLetterDTO>(`${BASE}/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`${BASE}/${id}`);
  },

  async duplicate(id: string): Promise<CoverLetterDTO> {
    return apiClient.post<CoverLetterDTO>(`${BASE}/${id}/duplicate`);
  },
};
