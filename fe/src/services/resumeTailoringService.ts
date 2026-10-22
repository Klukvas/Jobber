import { z } from "zod";
import {
  apiClient,
  ApiError,
  CLIENT_ERROR_CODES,
  type RequestOptions,
} from "./api";

const skillRef = z.object({ id: z.string().uuid(), name: z.string() });
const common = {
  id: z.string().min(1),
  reason: z.string(),
  evidence: z.array(
    z.object({
      section: z.enum(["summary", "experience", "skills"]),
      entry_id: z.string().optional(),
      quote: z.string(),
    }),
  ),
};
const changeSchema = z.discriminatedUnion("kind", [
  z.object({
    ...common,
    kind: z.literal("summary"),
    before: z.string(),
    after: z.string(),
  }),
  z.object({
    ...common,
    kind: z.literal("experience"),
    experience_id: z.string().uuid(),
    company: z.string(),
    position: z.string(),
    before: z.string(),
    after: z.string(),
  }),
  z.object({
    ...common,
    kind: z.literal("skill_order"),
    before: z.array(skillRef),
    after: z.array(skillRef),
  }),
]);
const proposalSchema = z.object({
  proposal_id: z.string().uuid(),
  expires_at: z.string().datetime({ offset: true }),
  source_resume: z.object({ id: z.string().uuid(), title: z.string() }),
  job: z.object({ id: z.string().uuid(), title: z.string() }),
  changes: z.array(changeSchema),
});
const appliedSchema = z.object({
  resume_builder: z.object({ id: z.string().uuid(), title: z.string() }),
  job_id: z.string().uuid(),
});
export type TailoringChange = z.infer<typeof changeSchema>;
export type TailoringProposal = z.infer<typeof proposalSchema>;
export type AppliedTailoring = z.infer<typeof appliedSchema>;
export type TailoringLocale = "en" | "ru" | "uk";

function validated<T>(schema: z.ZodType<T>, input: unknown): T {
  const result = schema.safeParse(input);
  if (!result.success) {
    throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
  }
  return result.data;
}

export const resumeTailoringService = {
  async generate(
    jobId: string,
    resumeId: string,
    locale: TailoringLocale,
    options?: RequestOptions,
  ): Promise<TailoringProposal> {
    const response = await apiClient.post<unknown>(
      `jobs/${jobId}/resume-tailoring`,
      { resume_builder_id: resumeId, locale },
      options,
    );
    const proposal = validated(proposalSchema, response);
    const uniqueChanges = new Set(proposal.changes.map((change) => change.id));
    if (
      proposal.job.id !== jobId ||
      proposal.source_resume.id !== resumeId ||
      uniqueChanges.size !== proposal.changes.length
    ) {
      throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
    }
    return proposal;
  },
  async apply(
    jobId: string,
    proposalId: string,
    acceptedChangeIds: readonly string[],
    title: string,
  ): Promise<AppliedTailoring> {
    const response = await apiClient.post<unknown>(
      `jobs/${jobId}/resume-tailoring/${proposalId}/apply`,
      { accepted_change_ids: acceptedChangeIds, title },
    );
    const applied = validated(appliedSchema, response);
    if (applied.job_id !== jobId) {
      throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0);
    }
    return applied;
  },
};
