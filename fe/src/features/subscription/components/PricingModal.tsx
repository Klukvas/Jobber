import { useTranslation } from "react-i18next";
import { Check, Info } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/Dialog";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { usePaddleCheckout } from "@/features/subscription/usePaddleCheckout";
import { FEATURES } from "@/shared/lib/features";
import type { SubscriptionPlan } from "@/shared/types/api";

interface PricingModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface PlanCardProps {
  name: string;
  price: string;
  features: string[];
  isCurrent: boolean;
  isHighlighted: boolean;
  onSelect: () => void;
  disabled: boolean;
  currentBadge: string;
  ctaLabel: string;
  popularLabel: string;
}

function PlanCard({
  name,
  price,
  features,
  isCurrent,
  isHighlighted,
  onSelect,
  disabled,
  currentBadge,
  ctaLabel,
  popularLabel,
}: PlanCardProps) {
  return (
    <div
      className={`relative flex flex-col rounded-xl border p-6 ${
        isHighlighted
          ? "border-transparent bg-primary/5 shadow-lg ring-2 ring-primary/50 dark:bg-primary/10 dark:ring-primary/40"
          : "border-border bg-card"
      }`}
    >
      {isHighlighted && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-primary px-3 py-0.5 text-xs font-medium text-primary-foreground">
          {popularLabel}
        </div>
      )}

      <div className="mb-4">
        <h3 className="text-lg font-semibold">{name}</h3>
        <p className="mt-2 text-3xl font-bold">{price}</p>
      </div>

      <ul className="mb-6 flex-1 space-y-3">
        {features.map((feature) => (
          <li key={feature} className="flex items-start gap-2 text-sm">
            <Check className="mt-0.5 h-4 w-4 shrink-0 text-green-500" />
            <span>{feature}</span>
          </li>
        ))}
      </ul>

      {isCurrent ? (
        <div className="rounded-md border border-border bg-muted px-4 py-2 text-center text-sm font-medium text-muted-foreground">
          {currentBadge}
        </div>
      ) : (
        <button
          className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition-all hover:bg-primary/90 hover:shadow-md disabled:cursor-not-allowed disabled:opacity-50"
          onClick={onSelect}
          disabled={disabled}
        >
          {ctaLabel}
        </button>
      )}
    </div>
  );
}

export function PricingModal({ open, onOpenChange }: PricingModalProps) {
  const { t } = useTranslation();
  const { plan } = useSubscription();
  const { openCheckout, isReady } = usePaddleCheckout();

  if (!FEATURES.PAYMENTS) return null;

  const handleSelect = (target: SubscriptionPlan) => {
    openCheckout(target);
    onOpenChange(false);
  };

  const plans = [
    {
      id: "free" as SubscriptionPlan,
      name: t("settings.subscription.freePlan"),
      price: t("settings.subscription.pricing.freePrice"),
      features: [
        t("settings.subscription.pricing.freeJobs"),
        t("settings.subscription.pricing.freeResumes"),
        t("settings.subscription.pricing.freeApplications"),
        t("settings.subscription.pricing.freeAI"),
        t("settings.subscription.pricing.freeJobParses"),
        t("settings.subscription.pricing.freeResumeBuilders"),
        t("settings.subscription.pricing.freeCoverLetters"),
      ],
      highlighted: false,
    },
    {
      id: "pro" as SubscriptionPlan,
      name: t("settings.subscription.proPlan"),
      price: t("settings.subscription.pricing.proPrice"),
      features: [
        t("settings.subscription.pricing.proJobs"),
        t("settings.subscription.pricing.proResumes"),
        t("settings.subscription.pricing.proApplications"),
        t("settings.subscription.pricing.proAI"),
        t("settings.subscription.pricing.proJobParses"),
        t("settings.subscription.pricing.proResumeBuilders"),
        t("settings.subscription.pricing.proCoverLetters"),
      ],
      highlighted: true,
    },
    {
      id: "enterprise" as SubscriptionPlan,
      name: t("settings.subscription.enterprisePlan"),
      price: t("settings.subscription.pricing.enterprisePrice"),
      features: [
        t("settings.subscription.pricing.enterpriseJobs"),
        t("settings.subscription.pricing.enterpriseResumes"),
        t("settings.subscription.pricing.enterpriseApplications"),
        t("settings.subscription.pricing.enterpriseAI"),
        t("settings.subscription.pricing.enterpriseJobParses"),
        t("settings.subscription.pricing.enterpriseResumeBuilders"),
        t("settings.subscription.pricing.enterpriseCoverLetters"),
      ],
      highlighted: false,
    },
  ];

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      className="max-w-[calc(100vw-2rem)] sm:max-w-4xl"
    >
      <DialogContent
        onClose={() => onOpenChange(false)}
        className="max-w-[calc(100vw-2rem)] sm:max-w-4xl"
      >
        <DialogHeader className="text-center">
          <DialogTitle className="text-center text-2xl">
            {t("settings.subscription.pricing.modalTitle")}
          </DialogTitle>
        </DialogHeader>

        {/* Payments disabled notice — only shown when payments are off */}
        {!FEATURES.PAYMENTS && (
          <div className="mt-4 flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950">
            <Info className="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
            <p className="text-sm text-amber-800 dark:text-amber-200">
              {t("settings.subscription.paymentsDisabled")}
            </p>
          </div>
        )}

        <div className="mt-6 grid grid-cols-1 gap-4 pt-4 md:grid-cols-3">
          {plans.map((p) => (
            <PlanCard
              key={p.id}
              name={p.name}
              price={p.price}
              features={p.features}
              isCurrent={plan === p.id}
              isHighlighted={p.highlighted}
              onSelect={() => handleSelect(p.id)}
              disabled={!FEATURES.PAYMENTS || !isReady}
              currentBadge={t("settings.subscription.pricing.currentPlanBadge")}
              ctaLabel={t("settings.subscription.pricing.choosePlan")}
              popularLabel={t("settings.subscription.pricing.popular")}
            />
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
