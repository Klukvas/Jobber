import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { FEATURES } from "@/shared/lib/features";
import { PricingModal } from "@/features/subscription/components/PricingModal";
import { useMonthlyResetDate } from "@/features/subscription/useMonthlyResetDate";

type UpgradeResource =
  | "jobs"
  | "resumes"
  | "ai"
  | "resume_builders"
  | "cover_letters";

interface UpgradeBannerProps {
  resource: UpgradeResource;
}

const limitKeyMap: Record<UpgradeResource, string> = {
  jobs: "limitReachedJobs",
  // Not "AI is not available on your plan": the free plan *has* one AI request
  // a month, shared by match scoring, resume assistance and resume imports —
  // and it comes back. Saying the feature is missing next to a reset date read
  // as two contradictory statements.
  ai: "limitReachedAIMonthly",
  resumes: "limitReachedResumes",
  resume_builders: "limitReachedResumeBuilders",
  cover_letters: "limitReachedCoverLetters",
};

const limitFieldMap: Record<
  UpgradeResource,
  | "max_jobs"
  | "max_resumes"
  | "max_ai_requests"
  | "max_resume_builders"
  | "max_cover_letters"
> = {
  jobs: "max_jobs",
  resumes: "max_resumes",
  ai: "max_ai_requests",
  resume_builders: "max_resume_builders",
  cover_letters: "max_cover_letters",
};

/**
 * The message for a resource, given the plan's cap for it.
 *
 * A cap of zero means the plan does not include the feature at all; any higher
 * cap means it does and it has been used up. Cover letters are the only
 * resource that is genuinely absent on one plan and merely finite on another,
 * so they are the only ones that need both sentences.
 */
function limitMessageKey(resource: UpgradeResource, limit: number): string {
  if (resource === "cover_letters" && limit > 0) {
    return "limitReachedCoverLettersUsed";
  }
  return limitKeyMap[resource];
}

/**
 * Resources whose usage the backend counts per calendar month, and which
 * therefore genuinely come back on their own (CountUserAIRequestsThisMonth).
 * Everything else — tracked jobs, resumes, resume builders, cover letters — is
 * a lifetime total (CountUserCoverLetters and friends), so promising a reset
 * date for those would be a lie.
 */
const MONTHLY_RESOURCES = new Set(["ai"]);

export function UpgradeBanner({ resource }: UpgradeBannerProps) {
  const { t } = useTranslation();
  const { limits, nextPlan } = useSubscription();
  const resetDate = useMonthlyResetDate();
  const [pricingOpen, setPricingOpen] = useState(false);

  if (!FEATURES.PAYMENTS || !nextPlan) return null;

  const limitValue = limits[limitFieldMap[resource]];
  const limitKey = limitMessageKey(resource, limitValue);

  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950">
      <p className="text-sm text-amber-800 dark:text-amber-200">
        {/* `count` drives i18next's plural selection, so "1 resume builder"
            reads correctly and RU/UA get their one/few/many forms. `limit` is
            kept for the two messages that have no number to agree with. */}
        {t(`settings.subscription.${limitKey}`, {
          count: limitValue,
          limit: limitValue,
        })}
      </p>
      {/* Monthly allowances come back on their own — saying when turns a dead
          end into a wait, and stops "upgrade" being the only way forward. */}
      {MONTHLY_RESOURCES.has(resource) && (
        <p className="mt-1 text-sm text-amber-700 dark:text-amber-300">
          {t("settings.subscription.resetsOn", { date: resetDate })}
        </p>
      )}
      <button
        type="button"
        onClick={() => setPricingOpen(true)}
        className="mt-3 inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      >
        {nextPlan === "enterprise"
          ? t("settings.subscription.upgradeToEnterprise")
          : t("settings.subscription.upgradeToPro")}
      </button>

      {/* Mounted on demand: the pricing modal pulls in the checkout hooks, and
          a banner that is merely visible should not be paying for them. */}
      {pricingOpen && <PricingModal open onOpenChange={setPricingOpen} />}
    </div>
  );
}
