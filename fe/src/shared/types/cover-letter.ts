// Cover Letter types

export interface CoverLetterDTO {
  id: string;
  resume_builder_id: string | null;
  job_id: string | null;
  title: string;
  template: string;
  recipient_name: string;
  recipient_title: string;
  company_name: string;
  company_address: string;
  greeting: string;
  paragraphs: string[];
  closing: string;
  font_family: string;
  font_size: number;
  primary_color: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCoverLetterRequest {
  title: string;
  resume_builder_id?: string | null;
  job_id?: string;
  template: string;
}

export interface UpdateCoverLetterRequest {
  title?: string;
  resume_builder_id?: string | null;
  job_id?: string | null;
  template?: string;
  recipient_name?: string;
  recipient_title?: string;
  company_name?: string;
  company_address?: string;
  greeting?: string;
  paragraphs?: string[];
  closing?: string;
  font_family?: string;
  font_size?: number;
  primary_color?: string;
}

export interface GenerateCoverLetterRequest {
  cover_letter_id: string;
  job_description?: string;
}

export interface GenerateCoverLetterResponse {
  greeting: string;
  paragraphs: string[];
  closing: string;
}
