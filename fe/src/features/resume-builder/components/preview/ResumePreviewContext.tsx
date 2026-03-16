import { createContext, useContext } from "react";
import type { FullResumeDTO } from "@/shared/types/resume-builder";

const ResumePreviewContext = createContext<FullResumeDTO | null>(null);

export const ResumePreviewProvider = ResumePreviewContext.Provider;

export function useResumePreview(): FullResumeDTO | null {
  return useContext(ResumePreviewContext);
}
