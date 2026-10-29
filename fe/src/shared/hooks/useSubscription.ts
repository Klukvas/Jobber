import { useQuery } from "@tanstack/react-query";
import { subscriptionService } from "@/services/subscriptionService";
import type {
  PlanLimits,
  SubscriptionPlan,
  SubscriptionUsage,
} from "@/shared/types/api";

export function useSubscription() {
  const { data: subscription, ...query } = useQuery({
    queryKey: ["subscription"],
    queryFn: subscriptionService.getSubscription,
    staleTime: 60_000,
  });

  const plan: SubscriptionPlan = subscription?.plan ?? "free";
  const isPro = plan === "pro" || plan === "enterprise";
  const isEnterprise = plan === "enterprise";
  const isFree = plan === "free";

  const nextPlan: SubscriptionPlan | null = isFree ? "pro" : null;

  const limits: PlanLimits = subscription?.limits ?? {
    max_jobs: 25,
    max_resumes: 1,
    max_ai_requests: 30,
    max_job_parses: 0,
    max_resume_builders: 2,
    max_cover_letters: 1,
  };

  const usage: SubscriptionUsage = subscription?.usage ?? {
    jobs: 0,
    resumes: 0,
    ai_requests: 0,
    job_parses: 0,
    resume_builders: 0,
    cover_letters: 0,
  };

  const canCreate = (
    resource: "jobs" | "resumes" | "ai" | "resume_builders" | "cover_letters",
  ): boolean => {
    switch (resource) {
      case "jobs":
        return limits.max_jobs < 0 || usage.jobs < limits.max_jobs;
      case "resumes":
        return limits.max_resumes < 0 || usage.resumes < limits.max_resumes;
      case "ai":
        if (subscription?.credits) return subscription.credits.remaining > 0;
        if (limits.max_ai_requests < 0) return true;
        if (limits.max_ai_requests === 0) return false;
        return usage.ai_requests < limits.max_ai_requests;
      case "resume_builders":
        return (
          limits.max_resume_builders < 0 ||
          usage.resume_builders < limits.max_resume_builders
        );
      case "cover_letters":
        if (limits.max_cover_letters === 0) return false;
        return (
          limits.max_cover_letters < 0 ||
          usage.cover_letters < limits.max_cover_letters
        );
      default:
        return true;
    }
  };

  return {
    subscription,
    credits: subscription?.credits,
    plan,
    isPro,
    isEnterprise,
    isFree,
    nextPlan,
    limits,
    usage,
    canCreate,
    ...query,
  };
}
