import { apiClient } from "./api";
import type {
  ContentLibraryEntryDTO,
  CreateContentLibraryRequest,
  UpdateContentLibraryRequest,
} from "@/shared/types/content-library";

const BASE = "content-library";

export const contentLibraryService = {
  async list(): Promise<ContentLibraryEntryDTO[]> {
    return apiClient.get<ContentLibraryEntryDTO[]>(BASE);
  },

  async create(
    data: CreateContentLibraryRequest,
  ): Promise<ContentLibraryEntryDTO> {
    return apiClient.post<ContentLibraryEntryDTO>(BASE, data);
  },

  async update(
    id: string,
    data: UpdateContentLibraryRequest,
  ): Promise<ContentLibraryEntryDTO> {
    return apiClient.patch<ContentLibraryEntryDTO>(`${BASE}/${id}`, data);
  },

  async remove(id: string): Promise<void> {
    return apiClient.delete<void>(`${BASE}/${id}`);
  },
};
