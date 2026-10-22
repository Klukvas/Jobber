import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { JobDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";
import type { PrepSession, PrepVersion } from "@/services/interviewPrepService";
import { ApiError } from "@/services/api";
import { InterviewPrepDialog } from "./InterviewPrepDialog";
const mocks = vi.hoisted(() => ({
  history: vi.fn(),
  operation: vi.fn(),
  version: vi.fn(),
  session: vi.fn(),
  generate: vi.fn(),
  studied: vi.fn(),
  start: vi.fn(),
  answer: vi.fn(),
  feedback: vi.fn(),
  action: vi.fn(),
  summary: vi.fn(),
  retry: vi.fn(),
}));
vi.mock("@/services/interviewPrepService", async (original) => ({
  ...(await original<typeof import("@/services/interviewPrepService")>()),
  interviewPrepService: mocks,
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }),
}));
const job: JobDTO = {
  id: "job-1",
  title: "Go Engineer",
  description: "Go services",
  is_archived: false,
  is_favorite: false,
  last_activity_at: "2026-10-22",
  created_at: "2026-10-22",
  updated_at: "2026-10-22",
  resume: { id: "resume-2", name: "Attached", type: "builder" },
};
const source = {
  id: "resume-2",
  title: "Attached Builder Resume",
} as ResumeBuilderDTO;
const other = {
  id: "resume-1",
  title: "Other Builder Resume",
} as ResumeBuilderDTO;
const question = {
  id: "q1",
  kind: "technical" as const,
  question: "Explain your approach",
  purpose: "Reasoning",
  difficulty: "medium" as const,
  advice: "<img src=x onerror=alert(1)>",
  evidence: [],
  star: null,
};
const version: PrepVersion = {
  id: "v1",
  job_id: job.id,
  resume_builder_id: source.id,
  resume_title: source.title,
  job_title: job.title,
  locale: "en",
  created_at: "2026-10-22T00:00:00Z",
  stale: false,
  studied: [],
  facts: [],
  plan: {
    requirements: [
      {
        requirement: "Go",
        assessment: "Clarify actual experience",
        evidence: [],
      },
    ],
    questions: [question],
    gaps: [],
    interviewer_questions: ["How does the team review code?"],
  },
};
const session: PrepSession = {
  answer_bytes_remaining: 90000,
  id: "s1",
  version_id: version.id,
  job_id: job.id,
  status: "active",
  revision: 0,
  questions: [question],
  answers: [],
  summary: null,
  created_at: "2026-10-22T00:00:00Z",
};
function setup(resumes: ResumeBuilderDTO[] = [other, source]) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
  const props = {
    job,
    builderResumes: resumes,
    isLoading: false,
    isError: false,
    isJobDirty: false,
    isJobSaving: false,
    onClose: vi.fn(),
  };
  return {
    ...render(<InterviewPrepDialog {...props} />, { wrapper }),
    client,
    user: userEvent.setup(),
    props,
  };
}
describe("saved interview preparation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.history.mockResolvedValue({
      versions: [version],
      sessions: [session],
      operations: [],
      ai_available: true,
    });
    mocks.version.mockResolvedValue(version);
    mocks.session.mockResolvedValue(session);
    mocks.start.mockResolvedValue(session);
    mocks.operation.mockImplementation(async (_job: string, id: string) => ({
      id,
      status: "failed",
      kind: "summary",
      session_id: session.id,
      error_code: "AI_FAILED",
      attempts: 1,
    }));
  });
  it("selects the attached Builder Resume after a delayed list arrives", async () => {
    const view = setup([]);
    expect(screen.getByLabelText("jobs.interview.source")).toHaveValue("");
    view.rerender(
      <InterviewPrepDialog {...view.props} builderResumes={[other, source]} />,
    );
    expect(screen.getByLabelText("jobs.interview.source")).toHaveValue(
      source.id,
    );
    await view.user.selectOptions(
      screen.getByLabelText("jobs.interview.source"),
      other.id,
    );
    view.rerender(
      <InterviewPrepDialog {...view.props} builderResumes={[source, other]} />,
    );
    expect(screen.getByLabelText("jobs.interview.source")).toHaveValue(
      other.id,
    );
  });
  it("renders AI text escaped inside the analytics exclusion boundary", async () => {
    setup();
    const advice = await screen.findByText(question.advice);
    expect(advice.closest(".ph-no-capture")).not.toBeNull();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });
  it("keeps saved preparation readable when AI is unavailable or quota fails", async () => {
    mocks.generate.mockRejectedValue(
      new ApiError("", "PLAN_LIMIT_REACHED", 403),
    );
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.regenerate" }),
    );
    await screen.findByText("jobs.interview.errors.PLAN_LIMIT_REACHED");
    expect(screen.getByText("Clarify actual experience")).toBeInTheDocument();
  });
  it("saves answers independently before offering feedback", async () => {
    mocks.answer.mockImplementation(async () => ({
      ...session,
      revision: 1,
      answers: [
        {
          id: "a1",
          question_id: "q1",
          text: "My real experience",
          skipped: false,
          feedback: null,
          created_at: "2026-10-22T00:00:00Z",
        },
      ],
    }));
    const { user, client } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", {
        name: "jobs.interview.practice",
      }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    await user.type(
      await screen.findByLabelText("jobs.interview.answer"),
      "My real experience",
    );
    mocks.session.mockResolvedValue({
      ...session,
      revision: 1,
      answers: [
        {
          id: "a1",
          question_id: "q1",
          text: "My real experience",
          skipped: false,
          feedback: null,
          created_at: "2026-10-22T00:00:00Z",
        },
      ],
    });
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.saveAnswer" }),
    );
    await screen.findByText("My real experience");
    expect(mocks.feedback).not.toHaveBeenCalled();
    expect(mocks.answer).toHaveBeenCalledOnce();
    expect(
      client.getQueryData(["interview-prep", job.id, "session", session.id]),
    ).toMatchObject({ answers: [{ text: "My real experience" }] });
  });
  it("prevents selection changes while a late session start response is pending", async () => {
    let resolve: (value: PrepSession) => void = () => {
      throw new Error("missing deferred resolver");
    };
    mocks.start.mockReturnValue(
      new Promise<PrepSession>((done) => {
        resolve = done;
      }),
    );
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", {
        name: "jobs.interview.practice",
      }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    await waitFor(() =>
      expect(screen.getByLabelText("jobs.interview.sessions")).toBeDisabled(),
    );
    expect(screen.getByLabelText("jobs.interview.history")).toBeDisabled();
    await act(async () => resolve(session));
    await waitFor(() =>
      expect(screen.getByLabelText("jobs.interview.sessions")).toBeEnabled(),
    );
  });
  it("reconciles a saved answer after a lost response without carrying its draft to the next question", async () => {
    const q2 = { ...question, id: "q2", question: "Second question" };
    const initial = { ...session, questions: [question, q2] };
    mocks.start.mockResolvedValue(initial);
    mocks.session.mockResolvedValue(initial);
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.practice" }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    await user.type(
      await screen.findByLabelText("jobs.interview.answer"),
      "Persisted first answer",
    );
    mocks.answer.mockImplementation(async () => {
      mocks.session.mockResolvedValue({
        ...initial,
        revision: 1,
        answers: [
          {
            id: "a1",
            question_id: "q1",
            text: "Persisted first answer",
            skipped: false,
            feedback: null,
            created_at: session.created_at,
          },
        ],
      });
      throw new Error("lost response after commit");
    });
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.saveAnswer" }),
    );
    await screen.findByText("Persisted first answer");
    await waitFor(() =>
      expect(screen.getByLabelText("jobs.interview.answer")).toHaveValue(""),
    );
    expect(
      screen.getByRole("heading", { name: "Second question" }),
    ).toBeInTheDocument();
  });
  it("retains an unsaved draft when starting another session is cancelled", async () => {
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.practice" }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    await user.type(
      await screen.findByLabelText("jobs.interview.answer"),
      "Keep my draft",
    );
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    expect(confirm).toHaveBeenCalled();
    expect(mocks.start).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText("jobs.interview.answer")).toHaveValue(
      "Keep my draft",
    );
    confirm.mockRestore();
  });
  it("preserves the local draft when serialized session capacity rejects a save", async () => {
    mocks.answer.mockRejectedValue(
      new ApiError("", "ANSWER_CAPACITY_EXCEEDED", 413),
    );
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.practice" }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    await user.type(
      await screen.findByLabelText("jobs.interview.answer"),
      "Keep this exact draft 😀",
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.saveAnswer" }),
    );
    await screen.findByText("jobs.interview.errors.ANSWER_CAPACITY_EXCEEDED");
    expect(screen.getByLabelText("jobs.interview.answer")).toHaveValue(
      "Keep this exact draft 😀",
    );
    expect(mocks.feedback).not.toHaveBeenCalled();
  });
  it("accepts 7000 emoji as 7000 characters and rejects an over-limit paste without truncation", async () => {
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.practice" }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    const input = await screen.findByLabelText("jobs.interview.answer");
    const unicode = "😀".repeat(7000);
    fireEvent.change(input, { target: { value: unicode } });
    expect(input).toHaveValue(unicode);
    expect(screen.getByText(/7000\/12000/)).toBeInTheDocument();
    fireEvent.change(input, { target: { value: "😀".repeat(12001) } });
    expect(input).toHaveValue(unicode);
    expect(
      screen.getByText("jobs.interview.errors.ANSWER_TOO_LONG"),
    ).toBeInTheDocument();
  });
  it("recovers an old failed summary outside history with explicit retry", async () => {
    const completed = { ...session, status: "completed" as const };
    mocks.start.mockResolvedValue(completed);
    mocks.session.mockResolvedValue(completed);
    const failed = {
      id: "old-summary",
      kind: "summary",
      status: "failed",
      error_code: "AI_FAILED",
      attempts: 1,
      session_id: session.id,
      answer_id: null,
      version_id: version.id,
      created_at: session.created_at,
    };
    mocks.summary.mockResolvedValue(failed);
    mocks.operation.mockResolvedValue(failed);
    mocks.retry.mockImplementation(async () => {
      mocks.operation.mockResolvedValue({
        ...failed,
        status: "succeeded",
        error_code: "",
      });
      mocks.session.mockResolvedValue({
        ...completed,
        summary: {
          criteria: [],
          strengths: [],
          improvements: ["Recovered summary"],
          revisit: [],
          followup: null,
        },
      });
      return { ...failed, status: "queued", error_code: "" };
    });
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.practice" }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    await user.click(
      await screen.findByRole("button", { name: "jobs.interview.getSummary" }),
    );
    expect(mocks.summary).toHaveBeenCalledTimes(1);
    expect(mocks.retry).not.toHaveBeenCalled();
    await user.click(
      await screen.findByRole("button", { name: "jobs.interview.retrySafe" }),
    );
    expect(mocks.retry).toHaveBeenCalledWith(job.id, "old-summary");
    expect(mocks.summary).toHaveBeenCalledTimes(1);
    await screen.findByText("Recovered summary");
  });
  it("allows older operation pages even with fewer than 20 versions and sessions", async () => {
    mocks.history.mockResolvedValue({
      versions: [version],
      sessions: [session],
      ai_available: true,
      operations: Array.from({ length: 30 }, (_, index) => ({
        id: `operation-${index}`,
        status: "succeeded",
        kind: "plan",
      })),
    });
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.older" }),
    );
    await waitFor(() =>
      expect(mocks.history).toHaveBeenCalledWith(job.id, 20, expect.anything()),
    );
  });
  it("shows explicit retry for old feedback without automatically calling the provider", async () => {
    const answered = {
      ...session,
      revision: 1,
      answers: [
        {
          id: "old-answer",
          question_id: "q1",
          text: "Saved original answer",
          skipped: false,
          feedback: null,
          created_at: session.created_at,
        },
      ],
    };
    mocks.start.mockResolvedValue(answered);
    mocks.session.mockResolvedValue(answered);
    const failed = {
      id: "old-feedback",
      kind: "feedback",
      status: "failed",
      error_code: "AI_FAILED",
      attempts: 1,
      session_id: session.id,
      answer_id: "old-answer",
      version_id: version.id,
      created_at: session.created_at,
    };
    mocks.feedback.mockResolvedValue(failed);
    mocks.operation.mockResolvedValue(failed);
    const { user } = setup();
    await screen.findByText("Clarify actual experience");
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.practice" }),
    );
    await user.click(
      screen.getByRole("button", { name: "jobs.interview.start" }),
    );
    await user.click(
      await screen.findByRole("button", { name: "jobs.interview.getFeedback" }),
    );
    expect(mocks.feedback).toHaveBeenCalledTimes(1);
    expect(mocks.retry).not.toHaveBeenCalled();
    expect(
      await screen.findByRole("button", { name: "jobs.interview.retrySafe" }),
    ).toBeEnabled();
  });
});
