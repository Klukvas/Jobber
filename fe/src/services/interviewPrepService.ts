import { z } from "zod";
import {
  apiClient,
  ApiError,
  CLIENT_ERROR_CODES,
  type RequestOptions,
} from "./api";

// Go validates UTF-8 runes; JS string.length measures UTF-16 code units.
export function interviewTextLength(value: string): number {
  return Array.from(value).length;
}
const text = z.string().refine((value) => interviewTextLength(value) <= 12000);
const evidence = z.object({ fact_id: text, quote: text });
const question = z.object({
  id: text,
  kind: z.enum(["technical", "behavioral", "situational", "followup"]),
  question: text,
  purpose: text,
  difficulty: z.enum(["easy", "medium", "hard"]),
  advice: text,
  evidence: z.array(evidence).max(8),
  star: z
    .object({ situation: text, task: text, action: text, result: text })
    .nullable(),
});
const review = z.object({
  criteria: z
    .array(
      z.object({
        name: z.enum(["content", "specificity", "structure"]),
        score: z.number().int().min(0).max(4),
        explanation: text,
        evidence: z.array(evidence).max(8),
      }),
    )
    .length(3),
  strengths: z.array(text).max(10),
  improvements: z.array(text).max(10),
  revisit: z.array(text).max(10),
  followup: z.object({ question: text, purpose: text }).nullable(),
});
const version = z.object({
  id: z.uuid(),
  job_id: z.uuid(),
  resume_builder_id: z.uuid(),
  resume_title: text,
  job_title: text,
  locale: z.enum(["en", "ru", "uk"]),
  created_at: z.string(),
  stale: z.boolean(),
  studied: z.array(text),
  facts: z.array(z.object({ id: text, section: text, text })),
  plan: z.object({
    requirements: z.array(
      z.object({
        requirement: text,
        assessment: text,
        evidence: z.array(evidence),
      }),
    ),
    questions: z.array(question).max(12),
    gaps: z.array(
      z.object({
        priority: z.enum(["high", "medium", "low"]),
        gap: text,
        action: text,
      }),
    ),
    interviewer_questions: z.array(text),
  }),
});
const operation = z.object({
  id: z.uuid(),
  kind: z.enum(["plan", "feedback", "summary"]),
  status: z.enum(["queued", "running", "succeeded", "failed", "interrupted"]),
  error_code: z.string(),
  attempts: z.number().int().min(0).max(3),
  created_at: z.string(),
  version_id: z.uuid().nullable(),
  session_id: z.uuid().nullable(),
  answer_id: z.uuid().nullable(),
});
const session = z.object({
  answer_bytes_remaining: z.number().int().nonnegative(),
  id: z.uuid(),
  version_id: z.uuid(),
  job_id: z.uuid(),
  status: z.enum(["active", "paused", "completed"]),
  revision: z.number().int().nonnegative(),
  questions: z.array(question).max(24),
  answers: z
    .array(
      z.object({
        id: z.uuid(),
        question_id: text,
        text,
        skipped: z.boolean(),
        feedback: review.nullable(),
        created_at: z.string(),
      }),
    )
    .max(24),
  summary: review.nullable(),
  created_at: z.string(),
});
const versionSummary = version.pick({
  id: true,
  job_id: true,
  resume_builder_id: true,
  resume_title: true,
  job_title: true,
  locale: true,
  created_at: true,
});
const sessionSummary = session.pick({
  id: true,
  version_id: true,
  job_id: true,
  status: true,
  revision: true,
  created_at: true,
});
const history = z.object({
  versions: z.array(versionSummary).max(20),
  sessions: z.array(sessionSummary).max(20),
  operations: z.array(operation).max(30),
  ai_available: z.boolean(),
});
export type PrepQuestion = z.infer<typeof question>;
export type PrepReview = z.infer<typeof review>;
export type PrepVersion = z.infer<typeof version>;
export type PrepSession = z.infer<typeof session>;
export type PrepOperation = z.infer<typeof operation>;
export type PrepHistory = z.infer<typeof history>;
export type PrepLocale = PrepVersion["locale"];
export type PrepEvidence = z.infer<typeof evidence>;

function parse<T>(schema: z.ZodType<T>, value: unknown): T {
  const result = schema.safeParse(value);
  if (!result.success)
    throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
  return result.data;
}
function path(job: string): string {
  return `jobs/${job}/interview-prep`;
}
function belongs<T extends { job_id: string }>(data: T, job: string): T {
  if (data.job_id !== job)
    throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
  return data;
}
export const interviewPrepService = {
  async history(
    job: string,
    offset = 0,
    options?: RequestOptions,
  ): Promise<PrepHistory> {
    const data = parse(
      history,
      await apiClient.get<unknown>(`${path(job)}?offset=${offset}`, options),
    );
    data.versions.forEach((v) => belongs(v, job));
    data.sessions.forEach((s) => belongs(s, job));
    return data;
  },
  async version(
    job: string,
    id: string,
    options?: RequestOptions,
  ): Promise<PrepVersion> {
    const data = belongs(
      parse(
        version,
        await apiClient.get<unknown>(`${path(job)}/versions/${id}`, options),
      ),
      job,
    );
    if (data.id !== id)
      throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
    return data;
  },
  async session(
    job: string,
    id: string,
    options?: RequestOptions,
  ): Promise<PrepSession> {
    const data = belongs(
      parse(
        session,
        await apiClient.get<unknown>(`${path(job)}/sessions/${id}`, options),
      ),
      job,
    );
    if (data.id !== id)
      throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
    return data;
  },
  async generate(
    job: string,
    resume: string,
    locale: PrepLocale,
    key: string,
  ): Promise<PrepOperation> {
    return parse(
      operation,
      await apiClient.post<unknown>(`${path(job)}/generate`, {
        resume_builder_id: resume,
        locale,
        key,
      }),
    );
  },
  async operation(
    job: string,
    id: string,
    options?: RequestOptions,
  ): Promise<PrepOperation> {
    const data = parse(
      operation,
      await apiClient.get<unknown>(`${path(job)}/operations/${id}`, options),
    );
    if (data.id !== id)
      throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
    return data;
  },
  async retry(job: string, id: string): Promise<PrepOperation> {
    return parse(
      operation,
      await apiClient.post<unknown>(`${path(job)}/operations/${id}/retry`, {}),
    );
  },
  async studied(
    job: string,
    id: string,
    questionId: string,
    studied: boolean,
  ): Promise<PrepVersion> {
    return belongs(
      parse(
        version,
        await apiClient.put<unknown>(`${path(job)}/versions/${id}/studied`, {
          question_id: questionId,
          studied,
        }),
      ),
      job,
    );
  },
  async start(
    job: string,
    versionId: string,
    key: string,
  ): Promise<PrepSession> {
    return belongs(
      parse(
        session,
        await apiClient.post<unknown>(`${path(job)}/sessions`, {
          version_id: versionId,
          key,
        }),
      ),
      job,
    );
  },
  async answer(
    job: string,
    id: string,
    body: {
      key: string;
      question_id: string;
      text: string;
      skip: boolean;
      revision: number;
    },
  ): Promise<PrepSession> {
    return belongs(
      parse(
        session,
        await apiClient.post<unknown>(
          `${path(job)}/sessions/${id}/answers`,
          body,
        ),
      ),
      job,
    );
  },
  async action(
    job: string,
    id: string,
    action: "pause" | "resume" | "finish",
    revision: number,
    key: string,
  ): Promise<PrepSession> {
    return belongs(
      parse(
        session,
        await apiClient.post<unknown>(`${path(job)}/sessions/${id}/action`, {
          key,
          action,
          revision,
        }),
      ),
      job,
    );
  },
  async feedback(
    job: string,
    sessionId: string,
    answerId: string,
    key: string,
  ): Promise<PrepOperation> {
    return parse(
      operation,
      await apiClient.post<unknown>(
        `${path(job)}/sessions/${sessionId}/answers/${answerId}/feedback`,
        { key },
      ),
    );
  },
  async summary(
    job: string,
    sessionId: string,
    key: string,
  ): Promise<PrepOperation> {
    return parse(
      operation,
      await apiClient.post<unknown>(
        `${path(job)}/sessions/${sessionId}/summary`,
        { key },
      ),
    );
  },
};
