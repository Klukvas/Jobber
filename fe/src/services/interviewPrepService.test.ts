import { beforeEach, describe, expect, it, vi } from "vitest";
import { interviewPrepService } from "./interviewPrepService";
import { apiClient } from "./api";
vi.mock("./api", async (original) => {
  const module = await original<typeof import("./api")>();
  return {
    ...module,
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
  };
});
const job = "11111111-1111-4111-8111-111111111111";
const otherJob = "22222222-2222-4222-8222-222222222222";
const version = "33333333-3333-4333-8333-333333333333";
const session = "44444444-4444-4444-8444-444444444444";
const key = "55555555-5555-4555-8555-555555555555";
const validSession = {
  answer_bytes_remaining: 90000,
  id: session,
  job_id: job,
  version_id: version,
  status: "active",
  revision: 0,
  questions: [],
  answers: [],
  summary: null,
  created_at: "2026-10-22T10:00:00Z",
};
describe("interview preparation contract", () => {
  beforeEach(() => vi.clearAllMocks());
  it("rejects malformed saved responses", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ versions: [] });
    await expect(interviewPrepService.history(job)).rejects.toMatchObject({
      code: "INVALID_RESPONSE",
    });
  });
  it("rejects a cross-Job late session response", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      ...validSession,
      job_id: otherJob,
    });
    await expect(
      interviewPrepService.session(job, session),
    ).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
  it("preserves answer retry identity and revision in save-only request", async () => {
    vi.mocked(apiClient.post).mockResolvedValue(validSession);
    const body = {
      key,
      question_id: "q1",
      text: "Saved real experience",
      skip: false,
      revision: 0,
    };
    await interviewPrepService.answer(job, session, body);
    expect(apiClient.post).toHaveBeenCalledWith(
      `jobs/${job}/interview-prep/sessions/${session}/answers`,
      body,
    );
    expect(apiClient.post).toHaveBeenCalledTimes(1);
  });
  it("supports unavailable AI while returning saved history", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      versions: [],
      sessions: [validSession],
      operations: [],
      ai_available: false,
    });
    const result = await interviewPrepService.history(job);
    expect(result.ai_available).toBe(false);
    expect(result.sessions).toHaveLength(1);
  });
  it("loads valid Unicode answers using backend code-point bounds", async () => {
    for (const length of [7000, 12000]) {
      const value = {
        ...validSession,
        answers: [
          {
            id: key,
            question_id: "q1",
            text: "😀".repeat(length),
            skipped: false,
            feedback: null,
            created_at: validSession.created_at,
          },
        ],
      };
      vi.mocked(apiClient.get).mockResolvedValue(value);
      expect(
        (await interviewPrepService.session(job, session)).answers[0].text,
      ).toBe(value.answers[0].text);
    }
    vi.mocked(apiClient.get).mockResolvedValue({
      ...validSession,
      answers: [
        {
          id: key,
          question_id: "q1",
          text: "😀".repeat(12001),
          skipped: false,
          feedback: null,
          created_at: validSession.created_at,
        },
      ],
    });
    await expect(
      interviewPrepService.session(job, session),
    ).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
  it("loads visible Unicode facts without rejecting surrogate pairs", async () => {
    const fact = "skill: " + "😀".repeat(7000);
    const value = {
      id: version,
      job_id: job,
      resume_builder_id: key,
      resume_title: "Resume",
      job_title: "Job",
      locale: "uk",
      created_at: validSession.created_at,
      stale: false,
      studied: [],
      facts: [{ id: "skills:known", section: "skills", text: fact }],
      plan: {
        requirements: [],
        questions: [],
        gaps: [],
        interviewer_questions: [],
      },
    };
    vi.mocked(apiClient.get).mockResolvedValue(value);
    expect(
      (await interviewPrepService.version(job, version)).facts[0].text,
    ).toBe(fact);
    vi.mocked(apiClient.get).mockResolvedValue({
      ...value,
      facts: [{ ...value.facts[0], text: "😀".repeat(12001) }],
    });
    await expect(
      interviewPrepService.version(job, version),
    ).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
  it("reads an operation receipt by identity without posting a retry", async () => {
    const receipt = {
      id: key,
      kind: "summary",
      status: "failed",
      error_code: "AI_FAILED",
      attempts: 1,
      created_at: validSession.created_at,
      session_id: session,
      version_id: version,
      answer_id: null,
    };
    vi.mocked(apiClient.get).mockResolvedValue(receipt);
    expect(await interviewPrepService.operation(job, key)).toEqual(receipt);
    expect(apiClient.post).not.toHaveBeenCalled();
    vi.mocked(apiClient.get).mockResolvedValue({ ...receipt, id: session });
    await expect(
      interviewPrepService.operation(job, key),
    ).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
});
