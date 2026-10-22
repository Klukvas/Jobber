import { beforeEach, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClientProvider } from "@tanstack/react-query";
import { RenameBuilderResumeModal } from "../RenameBuilderResumeModal";
import { getQueryClient } from "@/shared/lib/queryClient";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { enqueueResumeSave, markResumeLoaded, reconcileLoadedResume, resetResumeDocumentSync } from "@/features/resume-builder/lib/resumeDocumentSync";
import type { FullResumeDTO } from "@/shared/types/resume-builder";

vi.mock("react-i18next", () => ({useTranslation: () => ({t: (key: string) => key})}));
vi.mock("@/shared/lib/notifications", () => ({showSuccessNotification: vi.fn(), showErrorNotification: vi.fn()}));
vi.mock("@/services/resumeBuilderService", () => ({resumeBuilderService: {update: vi.fn(),upsertSummary: vi.fn()}}));
const initial: FullResumeDTO = {
 id:"rename-resume", title:"Old title", template_id:"modern", font_family:"Inter",primary_color:"#000",text_color:"#000",spacing:100,
 margin_top:40,margin_bottom:40,margin_left:40,margin_right:40,layout_mode:"single",sidebar_width:35,font_size:12,skill_display:"",created_at:"",updated_at:"",
 contact:null,summary:null,experiences:[],educations:[],skills:[],languages:[],certifications:[],projects:[],volunteering:[],custom_sections:[],section_order:[],
};
beforeEach(() => {
 vi.clearAllMocks(); resetResumeDocumentSync(); getQueryClient().clear();
 useResumeBuilderStore.getState().setResume(initial); markResumeLoaded(initial);
 getQueryClient().setQueryData(resumeQueryKeys.builderDetail(initial.id), initial);
 vi.mocked(resumeBuilderService.update).mockImplementation(async (_id, data) => ({...initial,...data}));
 vi.mocked(resumeBuilderService.upsertSummary).mockResolvedValue({content:"edited"});
});
it("keeps a list rename when a cached detail is reopened and another edit saves", async () => {
 const user = userEvent.setup();
 render(<QueryClientProvider client={getQueryClient()}><RenameBuilderResumeModal open onOpenChange={vi.fn()} resume={initial}/></QueryClientProvider>);
 const input = screen.getByLabelText(/resumes.titleLabel/);
 await user.clear(input); await user.type(input,"Renamed in list"); await user.click(screen.getByText("common.save"));
 await waitFor(() => expect(resumeBuilderService.update).toHaveBeenCalledWith(initial.id,{title:"Renamed in list"}));
 await waitFor(() => expect(getQueryClient().getQueryData<FullResumeDTO>(resumeQueryKeys.builderDetail(initial.id))?.title).toBe("Renamed in list"));
 const cached = getQueryClient().getQueryData<FullResumeDTO>(resumeQueryKeys.builderDetail(initial.id));
 if (!cached) throw new Error("missing detail cache");
 const loaded = reconcileLoadedResume(cached);
 useResumeBuilderStore.getState().setResume(loaded.resume); markResumeLoaded(loaded.resume);
 useResumeBuilderStore.getState().updateSummary({content:"edited"});
 await enqueueResumeSave();
 expect(vi.mocked(resumeBuilderService.update).mock.calls.at(-1)?.[1].title).toBe("Renamed in list");
});
