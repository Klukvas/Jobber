import { useTranslation } from "react-i18next";
import { useSubscription } from "@/shared/hooks/useSubscription";

export type PlanLimitResource =
  | "jobs"
  | "resumes"
  | "ai"
  | "resume_builders"
  | "cover_letters";

const RESOURCE_KEY: Record<PlanLimitResource, string> = {
  jobs: "planLimitResourceJobs",
  resumes: "planLimitResourceResumes",
  ai: "planLimitResourceAI",
  resume_builders: "planLimitResourceResumeBuilders",
  cover_letters: "planLimitResourceCoverLetters",
};

/**
 * The neutral, informational "you've hit your plan's cap" sentence used while
 * payments are switched off (there is no upgrade path to offer, but a disabled
 * control or a refused create must still say why). E.g. "You've reached the
 * limit of your current plan (25 of 25 tracked jobs)."
 *
 * Returns a function so one call site can ask about several resources (the
 * Resumes create menu has two).
 */
export function usePlanLimitMessage(): (resource: PlanLimitResource) => string {
  const { t } = useTranslation();
  const { limits, usage } = useSubscription();

  return (resource) => {
    const limit = {
      jobs: limits.max_jobs,
      resumes: limits.max_resumes,
      ai: limits.max_ai_requests,
      resume_builders: limits.max_resume_builders,
      cover_letters: limits.max_cover_letters,
    }[resource];
    const used = {
      jobs: usage.jobs,
      resumes: usage.resumes,
      ai: usage.ai_requests,
      resume_builders: usage.resume_builders,
      cover_letters: usage.cover_letters,
    }[resource];

    // A cap of zero is "not part of this plan", not "all zero used".
    if (resource === "cover_letters" && limit === 0) {
      return t("settings.subscription.planLimitCoverLettersNotIncluded");
    }

    return t("settings.subscription.planLimitInfo", {
      used,
      limit,
      resource: t(`settings.subscription.${RESOURCE_KEY[resource]}`, {
        count: limit,
      }),
    });
  };
}
