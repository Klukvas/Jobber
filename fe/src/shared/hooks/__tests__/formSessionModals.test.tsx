import { useState, type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CreateCompanyModal } from "@/features/companies/modals/CreateCompanyModal";
import { CreateJobModal } from "@/features/jobs/modals/CreateJobModal";
import { EditResumeModal } from "@/features/resumes/modals/EditResumeModal";
import { CreateStageTemplateModal } from "@/features/stages/modals/CreateStageTemplateModal";
import { ImportResumeModal } from "@/features/resume-builder/components/ImportResumeModal";
import type { CompanyDTO, JobDTO, ResumeDTO } from "@/shared/types/api";

const mocks = vi.hoisted(() => ({
  companyCreate: vi.fn(), companyUpdate: vi.fn(), jobCreate: vi.fn(), jobUpdate: vi.fn(), resumeUpdate: vi.fn(), stageCreate: vi.fn(), importText: vi.fn(), importPDF: vi.fn(),
  navigate: vi.fn(), success: vi.fn(), error: vi.fn(), close: vi.fn(), created: vi.fn(), markUnavailable: vi.fn(),
}));
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }) }));
vi.mock("react-router-dom", () => ({ useNavigate: () => mocks.navigate }));
vi.mock("@/services/companiesService", () => ({ companiesService: { create: mocks.companyCreate, update: mocks.companyUpdate, list: async () => ({ items: [] }) } }));
vi.mock("@/services/jobsService", () => ({ jobsService: { create: mocks.jobCreate, update: mocks.jobUpdate } }));
vi.mock("@/services/resumesService", () => ({ resumesService: { update: mocks.resumeUpdate, list: async () => ({ items: [] }) } }));
vi.mock("@/services/stageTemplatesService", () => ({ stageTemplatesService: { create: mocks.stageCreate } }));
vi.mock("@/services/resumeBuilderService", () => ({ resumeBuilderService: { list: async () => [], importFromText: mocks.importText, importFromPDF: mocks.importPDF } }));
vi.mock("@/shared/lib/notifications", () => ({ showSuccessNotification: mocks.success, showErrorNotification: mocks.error }));
vi.mock("@/shared/hooks/useSubscription", () => ({ useSubscription: () => ({ canCreate: () => true }) }));
vi.mock("@/shared/hooks/useAIAvailability", () => ({ useAIAvailability: () => ({ unavailable: false, isChecking: false, markUnavailable: mocks.markUnavailable }), aiErrorKey: () => "ai.requestFailed" }));
vi.mock("@/features/jobs/components/CompanySelectWithQuickAdd", () => ({ CompanySelectWithQuickAdd: () => null }));

const company = (id: string) => ({ id, name: `Company ${id}`, location: "", notes: "", is_favorite: false, applications_count: 0, active_applications_count: 0, derived_status: "idle", created_at: "2026-01-01", updated_at: "2026-01-01" }) as CompanyDTO;
const job = (id: string) => ({ id, title: `Job ${id}`, is_archived: false, is_favorite: false, created_at: "2026-01-01", updated_at: "2026-01-01" }) as JobDTO;
const resume = (id: string): ResumeDTO => ({ id, title: `Resume ${id}`, file_url: "", storage_type: "external", is_active: true, can_delete: true, applications_count: 0, created_at: "2026-01-01", updated_at: "2026-01-01" });
type FormProps = { open: boolean; onOpenChange: (open: boolean) => void; record: string };
const cases: { name: string; form: (props: FormProps) => ReactElement; request: typeof mocks.companyCreate; label: RegExp; submit: string; key: string; edit?: boolean }[] = [
  { name: "Company create", form: (p) => <CreateCompanyModal {...p} />, request: mocks.companyCreate, label: /companies.name/, submit: "common.create", key: "companies" },
  { name: "Company edit", form: (p) => <CreateCompanyModal {...p} company={company(p.record)} />, request: mocks.companyUpdate, label: /companies.name/, submit: "common.save", key: "companies", edit: true },
  { name: "Job create", form: (p) => <CreateJobModal {...p} onCreated={mocks.created} />, request: mocks.jobCreate, label: /jobs.title_field/, submit: "common.create", key: "jobs" },
  { name: "Job edit", form: (p) => <CreateJobModal {...p} job={job(p.record)} />, request: mocks.jobUpdate, label: /jobs.title_field/, submit: "common.save", key: "jobs", edit: true },
  { name: "Resume edit", form: (p) => <EditResumeModal {...p} resume={resume(p.record)} />, request: mocks.resumeUpdate, label: /resumes.titleLabel/, submit: "common.save", key: "resumes", edit: true },
  { name: "Stage create", form: (p) => <CreateStageTemplateModal {...p} initialOrder={0} />, request: mocks.stageCreate, label: /stages.stageName/, submit: "common.create", key: "stage-templates" },
];
function deferred() {
  let resolve!: (value: { id: string }) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<{ id: string }>((done, fail) => { resolve = done; reject = fail; });
  return { promise, resolve, reject };
}
function newClient() { return new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } }); }
beforeEach(() => vi.clearAllMocks());
describe.each(cases)("$name abandoned requests", (testCase) => {
  it.each(["success", "failure"])("%s from a cancelled session cannot affect a reopened draft", async (outcome) => {
    const user = userEvent.setup();
    const request = deferred(); testCase.request.mockReturnValue(request.promise);
    const client = newClient(); const invalidate = vi.spyOn(client, "invalidateQueries");
    function Harness() {
      const [open, setOpen] = useState(true);
      return <><button onClick={() => setOpen(true)}>Reopen</button>{testCase.form({ open, record: "a", onOpenChange: (value) => { mocks.close(value); setOpen(value); } })}</>;
    }
    render(<QueryClientProvider client={client}><Harness /></QueryClientProvider>);
    await user.clear(screen.getByLabelText(testCase.label)); await user.type(screen.getByLabelText(testCase.label), "Submitted draft A");
    await user.click(screen.getByRole("button", { name: testCase.submit }));
    await waitFor(() => expect(testCase.request).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole("button", { name: "common.cancel" }));
    await user.click(screen.getByRole("button", { name: "Reopen" }));
    await user.clear(screen.getByLabelText(testCase.label)); await user.type(screen.getByLabelText(testCase.label), "New session draft B");
    mocks.close.mockClear();
    await act(async () => { if (outcome === "success") request.resolve({ id: "created-a" }); else request.reject(new Error("abandoned failure")); });
    await waitFor(() => expect(client.getMutationCache().getAll()[0].state.status).toBe(outcome === "success" ? "success" : "error"));
    expect(screen.getByLabelText(testCase.label)).toHaveValue("New session draft B");
    expect(mocks.close).not.toHaveBeenCalled(); expect(mocks.created).not.toHaveBeenCalled(); expect(mocks.success).not.toHaveBeenCalled(); expect(mocks.error).not.toHaveBeenCalled();
    if (outcome === "success") expect(invalidate).toHaveBeenCalledWith({ queryKey: [testCase.key] });
  });
  if (testCase.edit) it("a pending record A response cannot close record B", async () => {
    const user = userEvent.setup(); const request = deferred(); testCase.request.mockReturnValue(request.promise);
    const client = newClient();
    const view = render(<QueryClientProvider client={client}>{testCase.form({ open: true, record: "a", onOpenChange: mocks.close })}</QueryClientProvider>);
    await user.click(screen.getByRole("button", { name: testCase.submit }));
    await waitFor(() => expect(testCase.request).toHaveBeenCalledTimes(1));
    view.rerender(<QueryClientProvider client={client}>{testCase.form({ open: true, record: "b", onOpenChange: mocks.close })}</QueryClientProvider>);
    await user.clear(screen.getByLabelText(testCase.label)); await user.type(screen.getByLabelText(testCase.label), "Record B draft");
    await act(async () => request.resolve({ id: "a" }));
    await waitFor(() => expect(client.getMutationCache().getAll()[0].state.status).toBe("success"));
    expect(screen.getByLabelText(testCase.label)).toHaveValue("Record B draft"); expect(mocks.close).not.toHaveBeenCalled(); expect(mocks.success).not.toHaveBeenCalled();
  });
});

it.each(["text", "pdf"])("abandoned %s import cannot reset a replacement form or navigate after leaving the route", async (format) => {
  const user = userEvent.setup(); const request = deferred();
  (format === "text" ? mocks.importText : mocks.importPDF).mockReturnValue(request.promise);
  const client = newClient(); const invalidate = vi.spyOn(client, "invalidateQueries");
  const view = render(<QueryClientProvider client={client}><ImportResumeModal open onOpenChange={mocks.close} /></QueryClientProvider>);
  if (format === "text") await user.type(screen.getByLabelText("resumeBuilder.import.resumeText"), "Synthetic import A");
  else {
    await user.click(screen.getByRole("button", { name: "resumeBuilder.import.uploadPdf" }));
    const input = view.container.querySelector<HTMLInputElement>('input[type="file"]');
    if (!input) throw new Error("file picker missing");
    await user.upload(input, new File(["%PDF-synthetic"], "synthetic.pdf", { type: "application/pdf" }));
  }
  await user.click(screen.getByRole("button", { name: "resumeBuilder.import.importButton" }));
  await waitFor(() => expect(format === "text" ? mocks.importText : mocks.importPDF).toHaveBeenCalledTimes(1));
  view.unmount();
  render(<QueryClientProvider client={client}><ImportResumeModal open onOpenChange={mocks.close} /></QueryClientProvider>);
  await user.type(screen.getByLabelText("resumeBuilder.import.resumeText"), "Replacement route draft");
  await act(async () => request.resolve({ id: "imported-a" }));
  await waitFor(() => expect(invalidate).toHaveBeenCalledWith({ queryKey: ["resume-builders"] }));
  expect(screen.getByLabelText("resumeBuilder.import.resumeText")).toHaveValue("Replacement route draft");
  expect(mocks.close).not.toHaveBeenCalled(); expect(mocks.navigate).not.toHaveBeenCalled(); expect(mocks.success).not.toHaveBeenCalled();
});
it("a rejected import after route unmount does not announce or change capability", async () => {
  const user = userEvent.setup(); const request = deferred(); mocks.importText.mockReturnValue(request.promise);
  const view = render(<QueryClientProvider client={newClient()}><ImportResumeModal open onOpenChange={mocks.close} /></QueryClientProvider>);
  await user.type(screen.getByLabelText("resumeBuilder.import.resumeText"), "Synthetic import");
  await user.click(screen.getByRole("button", { name: "resumeBuilder.import.importButton" }));
  view.unmount(); await act(async () => request.reject(new Error("abandoned")));
  expect(mocks.error).not.toHaveBeenCalled(); expect(mocks.markUnavailable).not.toHaveBeenCalled(); expect(mocks.navigate).not.toHaveBeenCalled(); expect(mocks.close).not.toHaveBeenCalled();
});
