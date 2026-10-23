import { markAIUnavailable } from "@/shared/hooks/useAIAvailability";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "@/services/api";
import {
  resumeTailoringService,
  type TailoringLocale,
  type TailoringProposal,
} from "@/services/resumeTailoringService";
import { ensureStoredResumeDocument } from "@/features/resume-builder/lib/resumeDocumentSync";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";

type Review = { proposal: TailoringProposal };
const STALE_CODES = new Set([
  "SOURCE_CHANGED", "JOB_CHANGED", "PROPOSAL_EXPIRED", "PROPOSAL_ALREADY_APPLIED",
]);

export function tailoringErrorKey(error: unknown): string {
  if (!(error instanceof ApiError)) return "requestFailed";
  if (STALE_CODES.has(error.code)) return "stale";
  if (error.code === "PLAN_LIMIT_REACHED") return "limitReached";
  if (error.status === 429) return "rateLimited";
  if (error.code === "AI_UNAVAILABLE") return "unavailable";
  if (error.code === "AI_TIMEOUT" || error.code === "TIMEOUT_ERROR") return "timeout";
  if (error.code === "AI_INVALID_RESPONSE" || error.code === "INVALID_RESPONSE") return "invalidResponse";
  if (error.code === "NOT_FOUND") return "notFound";
  return "requestFailed";
}

export function useResumeTailoring(jobId: string) {
  const queryClient = useQueryClient();
  const [review, setReview] = useState<Review | null>(null);
  const [acceptedIds, setAcceptedIds] = useState<readonly string[]>([]);
  const [isGenerating, setIsGenerating] = useState(false);
  const [errorKey, setErrorKey] = useState<string | null>(null);
  const [mustRegenerate, setMustRegenerate] = useState(false);
  const requestId = useRef(0);
  const controller = useRef<AbortController | null>(null);
  const applying = useRef(false);
  const generating = useRef(false);
  const mounted = useRef(true);
  const generateMutation = useMutation({
    mutationFn: (request: {
      resumeId: string;
      locale: TailoringLocale;
      signal: AbortSignal;
    }) => resumeTailoringService.generate(
      jobId, request.resumeId, request.locale, { signal: request.signal },
    ),
  });
  const applyMutation = useMutation({
    mutationFn: (request: {
      proposalId: string;
      ids: readonly string[];
      title: string;
    }) => resumeTailoringService.apply(
      jobId, request.proposalId, request.ids, request.title,
    ),
  });

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      requestId.current += 1;
      controller.current?.abort();
    };
  }, []);

  const reset = () => {
    requestId.current += 1;
    controller.current?.abort();
    generating.current = false;
    setIsGenerating(false);
    setReview(null);
    setAcceptedIds([]);
    setErrorKey(null);
    setMustRegenerate(false);
  };

  const generate = async (resumeId: string, locale: TailoringLocale) => {
    if (applying.current || generating.current) return;
    generating.current = true;
    const token = ++requestId.current;
    controller.current?.abort();
    const abort = new AbortController();
    controller.current = abort;
    setIsGenerating(true);
    setReview(null);
    setAcceptedIds([]);
    setErrorKey(null);
    setMustRegenerate(false);
    try {
      try {
        await ensureStoredResumeDocument(resumeId);
      } catch {
        if (mounted.current && token === requestId.current) {
          setErrorKey("sourceUnsaved");
        }
        return;
      }
      if (!mounted.current || token !== requestId.current) return;
      const proposal = await generateMutation.mutateAsync({
        resumeId, locale, signal: abort.signal,
      });
      if (!mounted.current || token !== requestId.current) return;
      setReview({ proposal });
    } catch (error) {
      if (mounted.current && token === requestId.current && !abort.signal.aborted) {
        markAIUnavailable(queryClient, error);
        setErrorKey(tailoringErrorKey(error));
      }
    } finally {
      void queryClient.invalidateQueries({ queryKey: ["subscription"] });
      if (mounted.current && token === requestId.current) {
        generating.current = false;
        setIsGenerating(false);
      }
    }
  };

  const apply = async (title: string, sourceId: string) => {
    if (
      applying.current || !review || acceptedIds.length === 0 ||
      !title.trim() || title.trim().length > 255 || mustRegenerate
    ) return null;
    if (
      sourceId !== review.proposal.source_resume.id ||
      Date.parse(review.proposal.expires_at) <= Date.now()
    ) {
      setErrorKey("stale");
      setMustRegenerate(true);
      return null;
    }
    applying.current = true;
    setErrorKey(null);
    try {
      const result = await applyMutation.mutateAsync({
        proposalId: review.proposal.proposal_id,
        ids: acceptedIds,
        title: title.trim(),
      });
      // Refresh all consumers; navigation must not wait for a Job refetch that
      // replaces this modal's saved-Job revision and unmounts it.
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: resumeQueryKeys.builderList() }),
        queryClient.invalidateQueries({ queryKey: ["job", jobId] }),
        queryClient.invalidateQueries({ queryKey: ["jobs"] }),
        queryClient.invalidateQueries({ queryKey: ["subscription"] }),
      ]);
      return mounted.current ? result : null;
    } catch (error) {
      if (mounted.current) {
        setErrorKey(tailoringErrorKey(error));
        if (error instanceof ApiError && STALE_CODES.has(error.code)) {
          setMustRegenerate(true);
        }
      }
      return null;
    } finally {
      applying.current = false;
    }
  };

  return {
    review,
    acceptedIds,
    isGenerating,
    isSaving: applyMutation.isPending,
    errorKey,
    mustRegenerate,
    reset,
    generate,
    apply,
    toggle: (id: string, checked: boolean) => setAcceptedIds((current) =>
      checked
        ? [...current.filter((value) => value !== id), id]
        : current.filter((value) => value !== id),
    ),
    selectAll: (checked: boolean) => setAcceptedIds(
      checked ? review?.proposal.changes.map((change) => change.id) ?? [] : [],
    ),
  };
}
