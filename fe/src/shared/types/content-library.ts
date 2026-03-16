export interface ContentLibraryEntryDTO {
  id: string;
  title: string;
  content: string;
  category: string;
  created_at: string;
  updated_at: string;
}

export interface CreateContentLibraryRequest {
  title: string;
  content: string;
  category: string;
}

export interface UpdateContentLibraryRequest {
  title?: string;
  content?: string;
  category?: string;
}

export type ContentLibraryCategory =
  | "bullet"
  | "summary"
  | "paragraph"
  | "other";

export const CONTENT_LIBRARY_CATEGORIES: ContentLibraryCategory[] = [
  "bullet",
  "summary",
  "paragraph",
  "other",
];
