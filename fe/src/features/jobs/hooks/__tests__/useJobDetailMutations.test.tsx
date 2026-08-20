import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";

const mockNavigate = vi.hoisted(() => vi.fn());

vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockJobsService = vi.hoisted(() => ({
  update: vi.fn(),
  toggleFavorite: vi.fn(),
  move: vi.fn(),
  archive: vi.fn(),
  unarchive: vi.fn(),
  delete: vi.fn(),
  updateStage: vi.fn(),
}));

const mockCommentsService = vi.hoisted(() => ({
  create: vi.fn(),
}));

const mockMatchScoreService = vi.hoisted(() => ({
  checkMatch: vi.fn(),
}));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/services/jobsService", () => ({ jobsService: mockJobsService }));
vi.mock("@/services/commentsService", () => ({
  commentsService: mockCommentsService,
}));
vi.mock("@/services/matchScoreService", () => ({
  matchScoreService: mockMatchScoreService,
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

// Real ApiError so `instanceof` checks in the hook execute.
vi.mock("@/services/api", () => ({
  ApiError: class ApiError extends Error {
    code: string;
    status: number;
    constructor(message: string, code: string, status: number) {
      super(message);
      this.name = "ApiError";
      this.code = code;
      this.status = status;
    }
  },
}));

import { ApiError } from "@/services/api";
import {
  useJobDetailMutations,
  fieldsFromJob,
  hasChanges,
  resumeSelectValue,
  type EditableFields,
} from "../useJobDetailMutations";
import type { JobDTO, MatchScoreResponse } from "@/shared/types/api";

function makeJob(overrides: Partial<JobDTO> = {}): JobDTO {
  return {
    id: "job-1",
    title: "Engineer",
    company_id: "c1",
    url: "https://example.com",
    source: "LinkedIn",
    description: "desc",
    is_archived: false,
    is_favorite: false,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  } as JobDTO;
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
  const Wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
  return { Wrapper, invalidateSpy, queryClient };
}

function makeSetters() {
  return {
    setFields: vi.fn<(fields: EditableFields) => void>(),
    setNewComment: vi.fn<(value: string) => void>(),
    setMatchScore: vi.fn<(data: MatchScoreResponse | null) => void>(),
    setMatchScoreError: vi.fn<(message: string | null) => void>(),
    setIsPricingModalOpen: vi.fn<(open: boolean) => void>(),
  };
}

type Setters = ReturnType<typeof makeSetters>;

function renderMutations(
  setters: Setters,
  opts: { id?: string; effectiveMatchResumeId?: string } = {},
) {
  const { Wrapper, invalidateSpy } = createWrapper();
  const view = renderHook(
    () =>
      useJobDetailMutations({
        id: opts.id ?? "job-1",
        effectiveMatchResumeId: opts.effectiveMatchResumeId ?? "resume-1",
        ...setters,
      }),
    { wrapper: Wrapper },
  );
  return { ...view, invalidateSpy };
}

describe("useJobDetailMutations helpers", () => {
  it("fieldsFromJob maps job to editable fields with defaults for null", () => {
    const job = makeJob({
      company_id: null as never,
      url: null as never,
      source: null as never,
      description: null as never,
    });
    expect(fieldsFromJob(job)).toEqual({
      title: "Engineer",
      company_id: "",
      url: "",
      source: "",
      description: "",
    });
  });

  it("hasChanges detects a modified field", () => {
    const job = makeJob();
    const fields = fieldsFromJob(job);
    expect(hasChanges(fields, job)).toBe(false);
    expect(hasChanges({ ...fields, title: "Changed" }, job)).toBe(true);
    expect(hasChanges({ ...fields, description: "new" }, job)).toBe(true);
  });

  it("resumeSelectValue builds type:id string or empty", () => {
    expect(resumeSelectValue(makeJob({ resume: undefined }))).toBe("");
    expect(
      resumeSelectValue(
        makeJob({ resume: { type: "builder", id: "rb-1" } as never }),
      ),
    ).toBe("builder:rb-1");
  });
});

describe("useJobDetailMutations", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("updateMutation", () => {
    it("calls jobsService.update with normalized empty->undefined fields and invalidates caches", async () => {
      const updated = makeJob({ title: "Updated" });
      mockJobsService.update.mockResolvedValue(updated);
      const setters = makeSetters();
      const { result, invalidateSpy } = renderMutations(setters);

      await act(async () => {
        result.current.updateMutation.mutate({
          title: "Updated",
          company_id: "",
          url: "https://x.com",
          source: "",
          description: "",
        } as Partial<EditableFields>);
      });

      await waitFor(() =>
        expect(result.current.updateMutation.isSuccess).toBe(true),
      );

      expect(mockJobsService.update).toHaveBeenCalledWith("job-1", {
        title: "Updated",
        company_id: undefined,
        url: "https://x.com",
        source: undefined,
        description: undefined,
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ["job", "job-1"],
      });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["jobs"] });
      expect(setters.setFields).toHaveBeenCalledWith(fieldsFromJob(updated));
      expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
        "jobs.updateSuccess",
      );
    });

    it("surfaces the error message on failure", async () => {
      mockJobsService.update.mockRejectedValue(new Error("boom"));
      const setters = makeSetters();
      const { result } = renderMutations(setters);

      await act(async () => {
        result.current.updateMutation.mutate({ title: "x" });
      });

      await waitFor(() =>
        expect(result.current.updateMutation.isError).toBe(true),
      );
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "boom",
      );
    });
  });

  describe("changeResumeMutation", () => {
    it("attaches an uploaded resume via resume_id", async () => {
      mockJobsService.update.mockResolvedValue(makeJob());
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.changeResumeMutation.mutate("uploaded:up-9");
      });
      await waitFor(() =>
        expect(result.current.changeResumeMutation.isSuccess).toBe(true),
      );
      expect(mockJobsService.update).toHaveBeenCalledWith("job-1", {
        resume_id: "up-9",
      });
    });

    it("attaches a builder resume via resume_builder_id", async () => {
      mockJobsService.update.mockResolvedValue(makeJob());
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.changeResumeMutation.mutate("builder:rb-3");
      });
      await waitFor(() =>
        expect(result.current.changeResumeMutation.isSuccess).toBe(true),
      );
      expect(mockJobsService.update).toHaveBeenCalledWith("job-1", {
        resume_builder_id: "rb-3",
      });
    });

    it("clears the attached resume on empty string", async () => {
      mockJobsService.update.mockResolvedValue(makeJob());
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.changeResumeMutation.mutate("");
      });
      await waitFor(() =>
        expect(result.current.changeResumeMutation.isSuccess).toBe(true),
      );
      expect(mockJobsService.update).toHaveBeenCalledWith("job-1", {
        resume_id: "",
      });
    });
  });

  describe("toggleFavoriteMutation", () => {
    it("calls toggleFavorite and invalidates on success", async () => {
      mockJobsService.toggleFavorite.mockResolvedValue({ is_favorite: true });
      const { result, invalidateSpy } = renderMutations(makeSetters());

      await act(async () => {
        result.current.toggleFavoriteMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.toggleFavoriteMutation.isSuccess).toBe(true),
      );
      expect(mockJobsService.toggleFavorite).toHaveBeenCalledWith("job-1");
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["jobs"] });
    });

    it("shows a generic error on failure", async () => {
      mockJobsService.toggleFavorite.mockRejectedValue(new Error("nope"));
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.toggleFavoriteMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.toggleFavoriteMutation.isError).toBe(true),
      );
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "common.error",
      );
    });
  });

  describe("moveMutation", () => {
    it("moves the job to the target stage template", async () => {
      mockJobsService.move.mockResolvedValue(makeJob());
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.moveMutation.mutate("tpl-2");
      });
      await waitFor(() =>
        expect(result.current.moveMutation.isSuccess).toBe(true),
      );
      expect(mockJobsService.move).toHaveBeenCalledWith("job-1", {
        stage_template_id: "tpl-2",
      });
      expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
        "jobs.board.moveSuccess",
      );
    });
  });

  describe("archive / unarchive / delete navigation", () => {
    it("archive navigates to jobs list on success", async () => {
      mockJobsService.archive.mockResolvedValue(makeJob({ is_archived: true }));
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.archiveMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.archiveMutation.isSuccess).toBe(true),
      );
      expect(mockNavigate).toHaveBeenCalledWith("/app/jobs");
    });

    it("unarchive does not navigate", async () => {
      mockJobsService.unarchive.mockResolvedValue(makeJob());
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.unarchiveMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.unarchiveMutation.isSuccess).toBe(true),
      );
      expect(mockNavigate).not.toHaveBeenCalled();
      expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
        "jobs.unarchiveSuccess",
      );
    });

    it("delete navigates to jobs list on success", async () => {
      mockJobsService.delete.mockResolvedValue(undefined);
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.deleteMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.deleteMutation.isSuccess).toBe(true),
      );
      expect(mockNavigate).toHaveBeenCalledWith("/app/jobs");
    });

    it("archive surfaces a fixed error message on failure", async () => {
      mockJobsService.archive.mockRejectedValue(new Error("x"));
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.archiveMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.archiveMutation.isError).toBe(true),
      );
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "jobs.archiveError",
      );
      expect(mockNavigate).not.toHaveBeenCalled();
    });
  });

  describe("completeCurrentStageMutation", () => {
    it("marks the stage completed with an ISO completed_at timestamp", async () => {
      mockJobsService.updateStage.mockResolvedValue({});
      const { result } = renderMutations(makeSetters());

      await act(async () => {
        result.current.completeCurrentStageMutation.mutate("stage-7");
      });
      await waitFor(() =>
        expect(result.current.completeCurrentStageMutation.isSuccess).toBe(
          true,
        ),
      );

      expect(mockJobsService.updateStage).toHaveBeenCalledTimes(1);
      const [jobId, stageId, payload] =
        mockJobsService.updateStage.mock.calls[0];
      expect(jobId).toBe("job-1");
      expect(stageId).toBe("stage-7");
      expect(payload.status).toBe("completed");
      // completed_at is generated via new Date().toISOString()
      expect(payload.completed_at).toMatch(
        /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/,
      );
      expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
        "jobs.stageCompletedSuccess",
      );
    });
  });

  describe("addCommentMutation", () => {
    it("creates a comment and clears the input", async () => {
      mockCommentsService.create.mockResolvedValue({ id: "cm-1" });
      const setters = makeSetters();
      const { result } = renderMutations(setters);

      await act(async () => {
        result.current.addCommentMutation.mutate("hello");
      });
      await waitFor(() =>
        expect(result.current.addCommentMutation.isSuccess).toBe(true),
      );
      expect(mockCommentsService.create).toHaveBeenCalledWith({
        job_id: "job-1",
        content: "hello",
      });
      expect(setters.setNewComment).toHaveBeenCalledWith("");
    });
  });

  describe("checkMatchMutation", () => {
    it("rejects when there is no resume selected", async () => {
      const setters = makeSetters();
      const { result } = renderMutations(setters, {
        effectiveMatchResumeId: "",
      });

      await act(async () => {
        result.current.checkMatchMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.checkMatchMutation.isError).toBe(true),
      );
      expect(mockMatchScoreService.checkMatch).not.toHaveBeenCalled();
      expect(setters.setMatchScoreError).toHaveBeenCalledWith(
        "jobs.matchScore.noResume",
      );
    });

    it("stores the score and closes the pricing modal on success", async () => {
      const scoreData = {
        overall_score: 80,
        categories: [],
        missing_keywords: [],
        strengths: [],
        summary: "",
        from_cache: false,
      };
      mockMatchScoreService.checkMatch.mockResolvedValue(scoreData);
      const setters = makeSetters();
      const { result } = renderMutations(setters);

      await act(async () => {
        result.current.checkMatchMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.checkMatchMutation.isSuccess).toBe(true),
      );
      expect(mockMatchScoreService.checkMatch).toHaveBeenCalledWith(
        "job-1",
        "resume-1",
      );
      expect(setters.setMatchScore).toHaveBeenCalledWith(scoreData);
      expect(setters.setMatchScoreError).toHaveBeenCalledWith(null);
      expect(setters.setIsPricingModalOpen).toHaveBeenCalledWith(false);
    });

    it("opens the pricing modal when the plan limit is reached", async () => {
      mockMatchScoreService.checkMatch.mockRejectedValue(
        new ApiError("limit", "PLAN_LIMIT_REACHED", 402),
      );
      const setters = makeSetters();
      const { result, invalidateSpy } = renderMutations(setters);

      await act(async () => {
        result.current.checkMatchMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.checkMatchMutation.isError).toBe(true),
      );
      expect(setters.setIsPricingModalOpen).toHaveBeenCalledWith(true);
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ["subscription"],
      });
    });

    it.each([
      ["JOB_DESCRIPTION_EMPTY", "jobs.matchScore.noDescription"],
      ["RESUME_FILE_EMPTY", "jobs.matchScore.noResumeFile"],
      ["AI_NOT_CONFIGURED", "jobs.matchScore.aiNotAvailable"],
      ["SOMETHING_ELSE", "jobs.matchScore.error"],
    ])("maps ApiError code %s to the right message", async (code, message) => {
      mockMatchScoreService.checkMatch.mockRejectedValue(
        new ApiError("err", code, 400),
      );
      const setters = makeSetters();
      const { result } = renderMutations(setters);

      await act(async () => {
        result.current.checkMatchMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.checkMatchMutation.isError).toBe(true),
      );
      expect(setters.setMatchScoreError).toHaveBeenCalledWith(message);
    });

    it("falls back to the raw message for non-ApiError failures", async () => {
      mockMatchScoreService.checkMatch.mockRejectedValue(
        new Error("plain failure"),
      );
      const setters = makeSetters();
      const { result } = renderMutations(setters);

      await act(async () => {
        result.current.checkMatchMutation.mutate();
      });
      await waitFor(() =>
        expect(result.current.checkMatchMutation.isError).toBe(true),
      );
      expect(setters.setMatchScoreError).toHaveBeenCalledWith("plain failure");
    });
  });
});
