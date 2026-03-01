import { useTranslation } from "react-i18next";
import { useSubscription } from "@/shared/hooks/useSubscription";

interface UpgradeBannerProps {
  resource: "jobs" | "resumes" | "applications" | "ai";
}

const limitKeyMap: Record<string, string> = {
  jobs: "limitReachedJobs",
  resumes: "limitReachedResumes",
  applications: "limitReachedApplications",
  ai: "limitReachedAI",
};

const limitFieldMap: Record<
  string,
  "max_jobs" | "max_resumes" | "max_applications" | "max_ai_requests"
> = {
  jobs: "max_jobs",
  resumes: "max_resumes",
  applications: "max_applications",
  ai: "max_ai_requests",
};

export function UpgradeBanner({ resource }: UpgradeBannerProps) {
  const { t } = useTranslation();
  const { limits, nextPlan } = useSubscription();

  if (!nextPlan) return null;

  const limitKey = limitKeyMap[resource];
  const limitValue = limits[limitFieldMap[resource]];

  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950">
      <p className="text-sm text-amber-800 dark:text-amber-200 mb-3">
        {t(`settings.subscription.${limitKey}`, { limit: limitValue })}
      </p>
      <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
        {t("settings.subscription.paymentsDisabled")}
      </p>
    </div>
  );
}
