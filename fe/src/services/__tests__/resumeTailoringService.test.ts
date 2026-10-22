import { beforeEach, describe, expect, it, vi } from "vitest";
const post = vi.hoisted(() => vi.fn());
vi.mock("@/services/api", async (importOriginal) => ({ ...await importOriginal<typeof import("@/services/api")>(), apiClient: { post } }));
import { resumeTailoringService } from "../resumeTailoringService";
const jobId = "11111111-1111-4111-8111-111111111111";
const sourceId = "22222222-2222-4222-8222-222222222222";
const proposalId = "33333333-3333-4333-8333-333333333333";
const proposal = { proposal_id: proposalId, expires_at: "2026-10-22T23:00:00Z", job: { id: jobId, title: "Engineer" }, source_resume: { id: sourceId, title: "Source" }, changes: [{ id: "summary", kind: "summary", before: "Original", after: "Suggested", reason: "Focus", evidence: [{ section: "summary", quote: "Original" }] }] };
describe("resumeTailoringService", () => {
  beforeEach(() => vi.clearAllMocks());
  it("uses the flat contract and forwards caller cancellation", async () => {
    post.mockResolvedValue(proposal);
    const controller = new AbortController();
    expect(await resumeTailoringService.generate(jobId, sourceId, "uk", { signal: controller.signal })).toEqual(proposal);
    expect(post).toHaveBeenCalledWith(`jobs/${jobId}/resume-tailoring`, { resume_builder_id: sourceId, locale: "uk" }, { signal: controller.signal });
  });
  it.each([
    { ...proposal, job: { ...proposal.job, id: sourceId } },
    { ...proposal, source_resume: { ...proposal.source_resume, id: jobId } },
    { ...proposal, changes: [proposal.changes[0], proposal.changes[0]] },
    { ...proposal, changes: [{ ...proposal.changes[0], after: 12 }] },
    { data: proposal },
  ])("rejects mismatched or malformed proposals", async (response) => {
    post.mockResolvedValue(response);
    await expect(resumeTailoringService.generate(jobId, sourceId, "en")).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
  it("accepts existing experience entries with blank company and position", async () => {
    post.mockResolvedValue({ ...proposal, changes: [{ ...proposal.changes[0], kind: "experience", experience_id: sourceId, company: "", position: "" }] });
    const result = await resumeTailoringService.generate(jobId, sourceId, "en");
    expect(result.changes[0].kind).toBe("experience");
  });
  it("sends only the explicit selections and title to atomic apply", async () => {
    post.mockResolvedValue({ resume_builder: { id: sourceId, title: "Copy" }, job_id: jobId });
    await resumeTailoringService.apply(jobId, proposalId, ["summary"], "Copy");
    expect(post).toHaveBeenCalledWith(`jobs/${jobId}/resume-tailoring/${proposalId}/apply`, { accepted_change_ids: ["summary"], title: "Copy" });
  });
  it("rejects an apply response for another Job", async () => {
    post.mockResolvedValue({ resume_builder: { id: sourceId, title: "Copy" }, job_id: sourceId });
    await expect(resumeTailoringService.apply(jobId, proposalId, ["summary"], "Copy")).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
});
