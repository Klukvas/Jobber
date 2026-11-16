import { beforeEach, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CoverLetterAIPanel } from "../CoverLetterAIPanel";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { apiClient } from "@/services/api";
import { flushCoverLetterDocument } from "../../lib/coverLetterDocumentSync";
import { createMockResume } from "@/features/resume-builder/components/__tests__/testHelpers";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock("@/shared/hooks/useSubscription", () => ({ useSubscription: () => ({ canCreate: () => true }) }));
vi.mock("@/shared/hooks/useAIAvailability", () => ({ useAIAvailability: () => ({ unavailable: false, isChecking: false, markUnavailable: vi.fn() }), aiErrorKey: () => "errors.aiUnavailable" }));
vi.mock("@/services/api", () => ({ apiClient: { post: vi.fn() } }));
vi.mock("@/services/resumeBuilderService", () => ({ resumeBuilderService: { list: vi.fn(), getById: vi.fn() } }));
vi.mock("../../lib/coverLetterDocumentSync", () => ({ flushCoverLetterDocument: vi.fn() }));
vi.mock("@/services/jobsService", () => ({ jobsService: { list: async () => ({ items: [
  { id: "job-1", title: "First role", company_name: "Acme", description: "First job description" },
  { id: "job-2", title: "Second role", company_name: "Beta", description: "Second job description" },
] }) } }));

beforeEach(() => {
  vi.clearAllMocks();
  useCoverLetterStore.getState().setCoverLetter({ id: "letter", resume_builder_id: null, job_id: null, title: "Letter", template: "professional", recipient_name: "", recipient_title: "", company_name: "", company_address: "", greeting: "", paragraphs: [""], closing: "", font_family: "Georgia", font_size: 12, primary_color: "#2563eb", created_at: "", updated_at: "" });
  vi.mocked(resumeBuilderService.list).mockResolvedValue([createMockResume({ id: "resume", title: "Candidate" })]);
  vi.mocked(resumeBuilderService.getById).mockResolvedValue(createMockResume({ id: "resume", summary: { content: "Built Go APIs" } }));
  vi.mocked(flushCoverLetterDocument).mockResolvedValue(undefined);
  vi.mocked(apiClient.post).mockResolvedValue({ greeting: "Hello", paragraphs: ["Generated paragraph"], closing: "Regards" });
});
function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><CoverLetterAIPanel /></QueryClientProvider>);
}

it("blocks empty job and missing/empty candidate source before a paid request", async () => {
  mount();
  const user = userEvent.setup();
  const generate = screen.getByRole("button", { name: "coverLetter.ai.generate" });
  await user.click(generate);
  expect(screen.getByRole("alert")).toHaveTextContent("coverLetter.ai.jobDescriptionRequired");
  await user.type(screen.getByLabelText("coverLetter.ai.jobDescription"), "Build APIs");
  await user.click(generate);
  expect(screen.getByRole("alert")).toHaveTextContent("coverLetter.ai.resumeRequired");
  await screen.findByRole("option", { name: "Candidate" });
  await user.selectOptions(screen.getByRole("combobox", { name: "coverLetter.ai.resume" }), "resume");
  vi.mocked(resumeBuilderService.getById).mockResolvedValueOnce(createMockResume({ summary: null }));
  await user.click(generate);
  await screen.findByText("coverLetter.ai.resumeRequired");
  expect(apiClient.post).not.toHaveBeenCalled();
  expect(flushCoverLetterDocument).not.toHaveBeenCalled();
});

it("selects the same job using End/Enter as with a pointer and Escape closes the list", async () => {
  mount();
  const user = userEvent.setup();
  const picker = await screen.findByRole("combobox", { name: "coverLetter.ai.selectJobPlaceholder" });
  await user.click(picker);
  await user.keyboard("{End}");
  expect(screen.getByRole("option", { name: "Second role (Beta)" })).toHaveAttribute("aria-selected", "true");
  await user.keyboard("{Enter}");
  expect(screen.getByLabelText("coverLetter.ai.jobDescription")).toHaveValue("Second job description");
  expect(useCoverLetterStore.getState().coverLetter?.job_id).toBe("job-2");
  await user.clear(picker);
  await user.click(screen.getByRole("option", { name: "First role (Acme)" }));
  expect(screen.getByLabelText("coverLetter.ai.jobDescription")).toHaveValue("First job description");
  expect(useCoverLetterStore.getState().coverLetter?.job_id).toBe("job-1");
  await user.click(picker);
  await user.keyboard("{Escape}");
  expect(picker).toHaveAttribute("aria-expanded", "false");
  expect(picker).toHaveFocus();
});

it("generates from a valid source and preserves the paid draft when saving Apply fails", async () => {
  mount();
  const user = userEvent.setup();
  await screen.findByRole("option", { name: "Candidate" });
  await user.selectOptions(screen.getByRole("combobox", { name: "coverLetter.ai.resume" }), "resume");
  await user.type(screen.getByLabelText("coverLetter.ai.jobDescription"), "Build APIs");
  await user.click(screen.getByRole("button", { name: "coverLetter.ai.generate" }));
  await screen.findByText("Generated paragraph");
  expect(apiClient.post).toHaveBeenCalledExactlyOnceWith("cover-letters/ai/generate", { cover_letter_id: "letter", job_description: "Build APIs" }, expect.any(Object));
  vi.mocked(flushCoverLetterDocument).mockRejectedValueOnce(new Error("offline"));
  await user.click(screen.getByRole("button", { name: "coverLetter.ai.apply" }));
  await screen.findByText("coverLetter.saveFailed");
  expect(useCoverLetterStore.getState().coverLetter?.paragraphs).toEqual(["Generated paragraph"]);
  expect(screen.getByText("Generated paragraph")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "coverLetter.ai.apply" }));
  await waitFor(() => expect(screen.queryByText("Generated paragraph")).not.toBeInTheDocument());
  expect(apiClient.post).toHaveBeenCalledTimes(1);
});
