import { useState } from "react";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/services/api";
import type { JobDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";
import type { TailoringProposal } from "@/services/resumeTailoringService";
import { ResumeTailoringModal } from "./ResumeTailoringModal";
import { JobResumeTailoringCard } from "./JobResumeTailoringCard";
const mocks = vi.hoisted(() => ({ generate: vi.fn(), apply: vi.fn(), prepare: vi.fn(), success: vi.fn() }));
vi.mock("@/services/resumeTailoringService", () => ({ resumeTailoringService: { generate: mocks.generate, apply: mocks.apply } }));
vi.mock("@/features/resume-builder/lib/resumeDocumentSync", () => ({ ensureStoredResumeDocument: mocks.prepare }));
vi.mock("@/shared/lib/notifications", () => ({ showSuccessNotification: mocks.success }));
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }) }));
const job: JobDTO = { id: "job-1", title: "Engineer", description: "Build React interfaces", is_archived: false, is_favorite: false, last_activity_at: "2026-01-01", created_at: "2026-01-01", updated_at: "2026-01-02" };
const source = { id: "source-1", title: "My Builder Resume", updated_at: "2026-01-01" } as ResumeBuilderDTO;
const otherSource = { ...source, id: "source-2", title: "Second Builder Resume" };
const proposal: TailoringProposal = {
  proposal_id: "proposal-1", expires_at: "2099-01-01T00:00:00Z", job: { id: job.id, title: job.title }, source_resume: { id: source.id, title: source.title },
  changes: [
    { id: "summary", kind: "summary", before: "I build interfaces", after: "React interface engineer", reason: "Emphasizes the relevant work", evidence: [{ section: "summary", quote: "I build interfaces" }] },
    { id: "experience", kind: "experience", experience_id: "experience-1", company: "Acme", position: "Engineer", before: "Created interfaces", after: "Created React interfaces", reason: "Matches the Job", evidence: [] },
    { id: "skills", kind: "skill_order", before: [{ id: "a", name: "SQL" }, { id: "b", name: "React" }], after: [{ id: "b", name: "React" }, { id: "a", name: "SQL" }], reason: "Relevant skill first", evidence: [] },
  ],
};
function Location() { return <p data-testid="location">{useLocation().pathname}</p>; }
function Harness({ resumes = [source, otherSource] }: { resumes?: ResumeBuilderDTO[] }) {
  const [open, setOpen] = useState(true);
  return <>{open && <ResumeTailoringModal job={job} builderResumes={resumes} isLoading={false} isError={false} onClose={() => setOpen(false)} />}<Location /></>;
}
function setup(children = <Harness />) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const view = render(<QueryClientProvider client={client}><MemoryRouter>{children}</MemoryRouter></QueryClientProvider>);
  return { ...view, client, user: userEvent.setup() };
}
function deferred<T>() { let resolve: (value: T) => void = () => {}; const promise = new Promise<T>((done) => { resolve = done; }); return { promise, resolve }; }
async function generate(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "jobs.tailoring.generate" }));
  await screen.findByText("jobs.tailoring.review");
}
describe("Job resume tailoring", () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.prepare.mockResolvedValue(undefined); mocks.generate.mockResolvedValue(proposal); mocks.apply.mockResolvedValue({ resume_builder: { id: "copy-1", title: "Copy" }, job_id: job.id }); });
  it("requires explicit keyboard selection and applies only the accepted change before opening the new editor", async () => {
    const { user, client } = setup();
    const invalidation = vi.spyOn(client, "invalidateQueries");
    await generate(user);
    expect(screen.getByText("React interface engineer")).toBeInTheDocument();
    expect(screen.getByText("Emphasizes the relevant work")).toBeInTheDocument();
    expect(screen.getByText("SQL → React")).toBeInTheDocument();
    expect(screen.getAllByRole("checkbox").every((checkbox) => !(checkbox as HTMLInputElement).checked)).toBe(true);
    const save = screen.getByRole("button", { name: "jobs.tailoring.saveCopy" });
    expect(save).toBeDisabled();
    const summary = screen.getByRole("checkbox", { name: "jobs.tailoring.summary" });
    summary.focus();
    await user.keyboard(" ");
    expect(save).toBeEnabled();
    await user.clear(screen.getByLabelText("jobs.tailoring.copyTitle"));
    await user.type(screen.getByLabelText("jobs.tailoring.copyTitle"), "  New copy  ");
    await user.click(save);
    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/app/resume-builder/copy-1"));
    expect(mocks.apply).toHaveBeenCalledExactlyOnceWith(job.id, proposal.proposal_id, ["summary"], "New copy");
    expect(invalidation).toHaveBeenCalledWith({ queryKey: ["resume-builders"] });
    expect(invalidation).toHaveBeenCalledWith({ queryKey: ["job", job.id] });
    expect(mocks.success).toHaveBeenCalledWith("jobs.tailoring.saved");
  });
  it("cancels generation on close and ignores a late successful response", async () => {
    const pending = deferred<TailoringProposal>(); mocks.generate.mockReturnValue(pending.promise);
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.generate" }));
    await waitFor(() => expect(mocks.generate).toHaveBeenCalledOnce());
    const options = mocks.generate.mock.calls[0][3] as { signal: AbortSignal };
    await user.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(options.signal.aborted).toBe(true);
    await act(async () => pending.resolve(proposal));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(mocks.apply).not.toHaveBeenCalled();
  });
  it("drops an old source response and starts a clean review for a new source", async () => {
    const pending = deferred<TailoringProposal>(); mocks.generate.mockReturnValueOnce(pending.promise);
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.generate" }));
    await waitFor(() => expect(mocks.generate).toHaveBeenCalledOnce());
    await user.selectOptions(screen.getByLabelText("jobs.tailoring.source"), otherSource.id);
    await act(async () => pending.resolve(proposal));
    expect(screen.queryByText("jobs.tailoring.review")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "jobs.tailoring.generate" })).toBeEnabled();
  });
  it("does not request AI when the modal closes while source saving is pending", async () => {
    const pending = deferred<void>(); mocks.prepare.mockReturnValueOnce(pending.promise);
    const { user } = setup();
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.generate" }));
    await user.click(screen.getByRole("button", { name: "common.cancel" }));
    await act(async () => pending.resolve());
    expect(mocks.generate).not.toHaveBeenCalled();
  });
  it("drops late generation results when the saved Job revision changes", async () => {
    const pending = deferred<TailoringProposal>(); mocks.generate.mockReturnValueOnce(pending.promise);
    function RevisionHarness() {
      const [revision, setRevision] = useState(job.updated_at);
      return <><button onClick={() => setRevision("new-revision")}>Change saved Job</button><JobResumeTailoringCard job={{ ...job, updated_at: revision }} builderResumes={[source]} isLoading={false} isError={false} isJobDirty={false} isJobSaving={false} /></>;
    }
    const { user } = setup(<RevisionHarness />);
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.title" }));
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.generate" }));
    await waitFor(() => expect(mocks.generate).toHaveBeenCalledOnce());
    await user.click(screen.getByRole("button", { name: "Change saved Job" }));
    await act(async () => pending.resolve(proposal));
    expect(screen.queryByText("jobs.tailoring.review")).not.toBeInTheDocument();
  });
  it("requires regeneration when a proposal has expired", async () => {
    mocks.generate.mockResolvedValueOnce({ ...proposal, expires_at: "2000-01-01T00:00:00Z" });
    const { user } = setup(); await generate(user);
    await user.click(screen.getByRole("checkbox", { name: "jobs.tailoring.summary" }));
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.saveCopy" }));
    expect(screen.getByRole("alert")).toHaveTextContent("jobs.tailoring.errors.stale");
    expect(mocks.apply).not.toHaveBeenCalled();
  });
  it("keeps selections after quota errors and prevents duplicate saves", async () => {
    mocks.apply.mockRejectedValueOnce(new ApiError("internal provider detail", "PLAN_LIMIT_REACHED", 403));
    const { user } = setup(); await generate(user);
    await user.click(screen.getByRole("checkbox", { name: "jobs.tailoring.selectAll" }));
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.saveCopy" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("jobs.tailoring.errors.limitReached");
    expect(screen.getByRole("checkbox", { name: "jobs.tailoring.summary" })).toBeChecked();
    expect(screen.queryByText("internal provider detail")).not.toBeInTheDocument();
    const pending = deferred<{ resume_builder: { id: string }; job_id: string }>(); mocks.apply.mockReturnValue(pending.promise);
    await user.dblClick(screen.getByRole("button", { name: "jobs.tailoring.saveCopy" }));
    expect(mocks.apply).toHaveBeenCalledTimes(2);
    expect(screen.getByRole("button", { name: "common.cancel" })).toBeDisabled();
    expect(screen.getByLabelText("jobs.tailoring.source")).toBeDisabled();
    await act(async () => pending.resolve({ resume_builder: { id: "copy-1" }, job_id: job.id }));
  });
  it("does not navigate to an old copy after switching Jobs during save", async () => {
    const pending = deferred<{ resume_builder: { id: string }; job_id: string }>();
    mocks.apply.mockReturnValueOnce(pending.promise);
    function JobSwitchHarness() {
      const [jobId, setJobId] = useState(job.id);
      return <><button onClick={() => setJobId("job-2")}>Switch Job</button><JobResumeTailoringCard key={jobId} job={{ ...job, id: jobId }} builderResumes={[source]} isLoading={false} isError={false} isJobDirty={false} isJobSaving={false} /><Location /></>;
    }
    const { user, client } = setup(<JobSwitchHarness />);
    const invalidation = vi.spyOn(client, "invalidateQueries");
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.title" }));
    await generate(user);
    await user.click(screen.getByRole("checkbox", { name: "jobs.tailoring.summary" }));
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.saveCopy" }));
    expect(screen.getByRole("button", { name: "common.cancel" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "common.close" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Switch Job" }));
    await act(async () => pending.resolve({ resume_builder: { id: "old-copy" }, job_id: job.id }));
    expect(screen.getByTestId("location")).toHaveTextContent("/");
    expect(screen.getByTestId("location")).not.toHaveTextContent("old-copy");
    expect(mocks.success).not.toHaveBeenCalled();
    expect(invalidation).toHaveBeenCalledWith({ queryKey: ["job", job.id] });
  });
  it("requires regeneration after a server stale response", async () => {
    mocks.apply.mockRejectedValueOnce(new ApiError("", "SOURCE_CHANGED", 409));
    const { user } = setup(); await generate(user);
    await user.click(screen.getByRole("checkbox", { name: "jobs.tailoring.summary" }));
    await user.click(screen.getByRole("button", { name: "jobs.tailoring.saveCopy" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("jobs.tailoring.errors.stale");
    expect(screen.queryByRole("button", { name: "jobs.tailoring.saveCopy" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "jobs.tailoring.regenerate" })).toBeEnabled();
  });
  it("shows AI unavailable without producing a fake review", async () => {
    mocks.generate.mockRejectedValueOnce(new ApiError("", "AI_UNAVAILABLE", 503));
    const { user } = setup(); await user.click(screen.getByRole("button", { name: "jobs.tailoring.generate" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("jobs.tailoring.errors.unavailable");
    expect(screen.queryByText("jobs.tailoring.review")).not.toBeInTheDocument();
    expect(mocks.apply).not.toHaveBeenCalled();
  });
  it("blocks an unsaved source before requesting AI and offers its editor", async () => {
    mocks.prepare.mockRejectedValueOnce(new Error("unsaved"));
    const { user } = setup(); await user.click(screen.getByRole("button", { name: "jobs.tailoring.generate" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("jobs.tailoring.errors.sourceUnsaved");
    expect(screen.getByRole("link", { name: "jobs.tailoring.openSource" })).toHaveAttribute("href", "/app/resume-builder/source-1");
    expect(mocks.generate).not.toHaveBeenCalled();
  });
  it("never offers save for an empty change set", async () => {
    mocks.generate.mockResolvedValueOnce({ ...proposal, changes: [] });
    const { user } = setup(); await generate(user);
    expect(screen.getByText("jobs.tailoring.noChanges")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "jobs.tailoring.saveCopy" })).not.toBeInTheDocument();
  });
  it("guides users without Builder Resumes to the existing Resumes page", () => {
    setup(<Harness resumes={[]} />);
    expect(screen.getByText("jobs.tailoring.empty")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "jobs.tailoring.createResume" })).toHaveAttribute("href", "/app/resumes");
    expect(screen.getByRole("button", { name: "jobs.tailoring.generate" })).toBeDisabled();
  });
  it.each([{ isJobDirty: true, description: job.description, message: "saveJobFirst" }, { isJobDirty: false, description: " ", message: "missingDescription" }])("blocks unsaved Jobs and missing descriptions", ({ isJobDirty, description, message }) => {
    setup(<JobResumeTailoringCard job={{ ...job, description }} builderResumes={[source]} isLoading={false} isError={false} isJobDirty={isJobDirty} isJobSaving={false} />);
    expect(screen.getByText(`jobs.tailoring.${message}`)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "jobs.tailoring.title" })).toBeDisabled();
  });
});

vi.mock("@/shared/hooks/useAIAvailability", async (importOriginal) => ({
  ...await importOriginal<typeof import("@/shared/hooks/useAIAvailability")>(),
  useAIAvailability: () => ({ unavailable: false, isChecking: false, markUnavailable: vi.fn() }),
}));
