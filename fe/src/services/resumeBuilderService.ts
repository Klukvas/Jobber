import { apiClient } from "./api";
import type {
  ResumeBuilderDTO,
  FullResumeDTO,
  ContactDTO,
  SummaryDTO,
  ExperienceDTO,
  EducationDTO,
  SkillDTO,
  LanguageDTO,
  CertificationDTO,
  ProjectDTO,
  VolunteeringDTO,
  CustomSectionDTO,
  SectionOrderDTO,
  CreateResumeBuilderRequest,
  UpdateResumeBuilderRequest,
  UpsertContactRequest,
  UpsertSummaryRequest,
  BatchUpdateSectionOrderRequest,
} from "@/shared/types/resume-builder";

const BASE = "resume-builder";

export const resumeBuilderService = {
  // Resume Builder CRUD
  async list(): Promise<ResumeBuilderDTO[]> {
    return apiClient.get<ResumeBuilderDTO[]>(BASE);
  },

  async getById(id: string): Promise<FullResumeDTO> {
    return apiClient.get<FullResumeDTO>(`${BASE}/${id}`);
  },

  async create(data: CreateResumeBuilderRequest): Promise<ResumeBuilderDTO> {
    return apiClient.post<ResumeBuilderDTO>(BASE, data);
  },

  async update(
    id: string,
    data: UpdateResumeBuilderRequest,
  ): Promise<ResumeBuilderDTO> {
    return apiClient.patch<ResumeBuilderDTO>(`${BASE}/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`${BASE}/${id}`);
  },

  async duplicate(id: string): Promise<ResumeBuilderDTO> {
    return apiClient.post<ResumeBuilderDTO>(`${BASE}/${id}/duplicate`);
  },

  // 1:1 sections
  async upsertContact(
    id: string,
    data: UpsertContactRequest,
  ): Promise<ContactDTO> {
    return apiClient.put<ContactDTO>(`${BASE}/${id}/contact`, data);
  },

  async upsertSummary(
    id: string,
    data: UpsertSummaryRequest,
  ): Promise<SummaryDTO> {
    return apiClient.put<SummaryDTO>(`${BASE}/${id}/summary`, data);
  },

  async updateSectionOrder(
    id: string,
    data: BatchUpdateSectionOrderRequest,
  ): Promise<SectionOrderDTO[]> {
    return apiClient.put<SectionOrderDTO[]>(
      `${BASE}/${id}/section-order`,
      data,
    );
  },

  // 1:N section helpers
  async createSection<T>(
    id: string,
    section: string,
    data: unknown,
  ): Promise<T> {
    return apiClient.post<T>(`${BASE}/${id}/${section}`, data);
  },

  async updateSection<T>(
    id: string,
    section: string,
    entryId: string,
    data: unknown,
  ): Promise<T> {
    return apiClient.patch<T>(`${BASE}/${id}/${section}/${entryId}`, data);
  },

  async deleteSection(
    id: string,
    section: string,
    entryId: string,
  ): Promise<void> {
    return apiClient.delete<void>(`${BASE}/${id}/${section}/${entryId}`);
  },

  // Typed section methods
  async createExperience(
    id: string,
    data: Omit<ExperienceDTO, "id">,
  ): Promise<ExperienceDTO> {
    return this.createSection<ExperienceDTO>(id, "experiences", data);
  },

  async updateExperience(
    id: string,
    entryId: string,
    data: Partial<ExperienceDTO>,
  ): Promise<ExperienceDTO> {
    return this.updateSection<ExperienceDTO>(id, "experiences", entryId, data);
  },

  async deleteExperience(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "experiences", entryId);
  },

  async createEducation(
    id: string,
    data: Omit<EducationDTO, "id">,
  ): Promise<EducationDTO> {
    return this.createSection<EducationDTO>(id, "educations", data);
  },

  async updateEducation(
    id: string,
    entryId: string,
    data: Partial<EducationDTO>,
  ): Promise<EducationDTO> {
    return this.updateSection<EducationDTO>(id, "educations", entryId, data);
  },

  async deleteEducation(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "educations", entryId);
  },

  async createSkill(id: string, data: Omit<SkillDTO, "id">): Promise<SkillDTO> {
    return this.createSection<SkillDTO>(id, "skills", data);
  },

  async updateSkill(
    id: string,
    entryId: string,
    data: Partial<SkillDTO>,
  ): Promise<SkillDTO> {
    return this.updateSection<SkillDTO>(id, "skills", entryId, data);
  },

  async deleteSkill(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "skills", entryId);
  },

  async createLanguage(
    id: string,
    data: Omit<LanguageDTO, "id">,
  ): Promise<LanguageDTO> {
    return this.createSection<LanguageDTO>(id, "languages", data);
  },

  async updateLanguage(
    id: string,
    entryId: string,
    data: Partial<LanguageDTO>,
  ): Promise<LanguageDTO> {
    return this.updateSection<LanguageDTO>(id, "languages", entryId, data);
  },

  async deleteLanguage(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "languages", entryId);
  },

  async createCertification(
    id: string,
    data: Omit<CertificationDTO, "id">,
  ): Promise<CertificationDTO> {
    return this.createSection<CertificationDTO>(id, "certifications", data);
  },

  async updateCertification(
    id: string,
    entryId: string,
    data: Partial<CertificationDTO>,
  ): Promise<CertificationDTO> {
    return this.updateSection<CertificationDTO>(
      id,
      "certifications",
      entryId,
      data,
    );
  },

  async deleteCertification(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "certifications", entryId);
  },

  async createProject(
    id: string,
    data: Omit<ProjectDTO, "id">,
  ): Promise<ProjectDTO> {
    return this.createSection<ProjectDTO>(id, "projects", data);
  },

  async updateProject(
    id: string,
    entryId: string,
    data: Partial<ProjectDTO>,
  ): Promise<ProjectDTO> {
    return this.updateSection<ProjectDTO>(id, "projects", entryId, data);
  },

  async deleteProject(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "projects", entryId);
  },

  async createVolunteering(
    id: string,
    data: Omit<VolunteeringDTO, "id">,
  ): Promise<VolunteeringDTO> {
    return this.createSection<VolunteeringDTO>(id, "volunteering", data);
  },

  async updateVolunteering(
    id: string,
    entryId: string,
    data: Partial<VolunteeringDTO>,
  ): Promise<VolunteeringDTO> {
    return this.updateSection<VolunteeringDTO>(
      id,
      "volunteering",
      entryId,
      data,
    );
  },

  async deleteVolunteering(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "volunteering", entryId);
  },

  async createCustomSection(
    id: string,
    data: Omit<CustomSectionDTO, "id">,
  ): Promise<CustomSectionDTO> {
    return this.createSection<CustomSectionDTO>(id, "custom-sections", data);
  },

  async updateCustomSection(
    id: string,
    entryId: string,
    data: Partial<CustomSectionDTO>,
  ): Promise<CustomSectionDTO> {
    return this.updateSection<CustomSectionDTO>(
      id,
      "custom-sections",
      entryId,
      data,
    );
  },

  async deleteCustomSection(id: string, entryId: string): Promise<void> {
    return this.deleteSection(id, "custom-sections", entryId);
  },

  async importFromText(data: {
    text: string;
    title?: string;
  }): Promise<FullResumeDTO> {
    return apiClient.post<FullResumeDTO>(`${BASE}/import/text`, data);
  },

  async importFromPDF(file: File, title?: string): Promise<FullResumeDTO> {
    const formData = new FormData();
    formData.append("file", file);
    if (title) {
      formData.append("title", title);
    }
    return apiClient.postFormData<FullResumeDTO>(
      `${BASE}/import/pdf`,
      formData,
    );
  },
};
