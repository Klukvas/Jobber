import { useTranslation } from "react-i18next";
import { Check, X } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/Dialog";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { usePlanSelection } from "@/features/subscription/usePlanSelection";
import { FEATURES } from "@/shared/lib/features";
import type { SubscriptionPlan } from "@/shared/types/api";

interface PricingModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface PlanFeature {
  readonly label: string;
  /** false renders a cross — the line describes a limitation, not a feature. */
  readonly included: boolean;
}

interface PlanCardProps {
  name: string;
  price: string;
  features: PlanFeature[];
  isCurrent: boolean;
  isHighlighted: boolean;
  onSelect: () => void;
  disabled: boolean;
  /** Replaces the CTA when this plan cannot be bought from here at all. */
  unavailableNote?: string;
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
  unavailableNote,
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
        {features.map(({ label, included }) => (
          <li
            key={label}
            className={`flex items-start gap-2 text-sm ${
              included ? "" : "text-muted-foreground"
            }`}
          >
            {included ? (
              <Check
                aria-hidden="true"
                className="mt-0.5 h-4 w-4 shrink-0 text-green-500"
              />
            ) : (
              <X
                aria-hidden="true"
                className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
              />
            )}
            <span>{label}</span>
          </li>
        ))}
      </ul>

      {isCurrent ? (
        <div className="rounded-md border border-border bg-muted px-4 py-2 text-center text-sm font-medium text-muted-foreground">
          {currentBadge}
        </div>
      ) : unavailableNote ? (
        // No button at all rather than a disabled one: this plan is not bought,
        // it is reached by cancelling, and a dead CTA would only look broken.
        <p className="px-4 py-2 text-center text-sm text-muted-foreground">
          {unavailableNote}
        </p>
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
  const {
    selectPlan,
    canSelect,
    isSubscriber,
    isReady,
    isPending,
    isCheckoutPopupOpen,
    errorMessageKey,
  } = usePlanSelection({ onPlanChanged: () => onOpenChange(false) });

  if (!FEATURES.PAYMENTS) return null;

  // A free user gets the provider's checkout popup drawn over this page —
  // nothing navigates, and no URL is involved. The modal stays open behind it
  // on purpose: if starting the checkout failed, this is where the user sees
  // why. A subscriber changes plan in place and the modal closes on success.
  const handleSelect = (target: SubscriptionPlan) => {
    selectPlan(target);
  };

  const included = (key: string): PlanFeature => ({
    label: t(`settings.subscription.pricing.${key}`),
    included: true,
  });

  const plans = [
    {
      id: "free" as SubscriptionPlan,
      name: t("settings.subscription.freePlan"),
      price: t("settings.subscription.pricing.freePrice"),
      features: [
        included("freeJobs"),
        included("freeResumes"),
        included("freeAI"),
        included("freeJobParses"),
        included("freeResumeBuilders"),
        // Free has no cover letters at all — a green check here reads as a
        // feature the plan does not have.
        {
          label: t("settings.subscription.pricing.freeCoverLetters"),
          included: false,
        },
      ],
      highlighted: false,
    },
    {
      id: "pro" as SubscriptionPlan,
      name: t("settings.subscription.proPlan"),
      price: t("settings.subscription.pricing.proPrice"),
      features: [
        included("proJobs"),
        included("proResumes"),
        included("proAI"),
        included("proJobParses"),
        included("proResumeBuilders"),
        included("proCoverLetters"),
      ],
      highlighted: true,
    },
    {
      id: "enterprise" as SubscriptionPlan,
      name: t("settings.subscription.enterprisePlan"),
      price: t("settings.subscription.pricing.enterprisePrice"),
      features: [
        included("enterpriseJobs"),
        included("enterpriseResumes"),
        included("enterpriseAI"),
        included("enterpriseJobParses"),
        included("enterpriseResumeBuilders"),
        included("enterpriseCoverLetters"),
      ],
      highlighted: false,
    },
  ];

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      className="max-w-[calc(100vw-2rem)] sm:max-w-4xl"
      // The payment popup is appended outside this dialog: while it is up, the
      // keyboard has to reach it, and Escape must not close the page behind a
      // payment in flight.
      hasExternalOverlay={isCheckoutPopupOpen}
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

        {errorMessageKey && (
          <p role="alert" className="mt-4 text-sm text-destructive">
            {t(errorMessageKey)}
          </p>
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
              disabled={!FEATURES.PAYMENTS || !isReady || isPending}
              // Downgrading to free means cancelling the subscription, which
              // lives in the manage flow — so this card offers no CTA here.
              unavailableNote={
                canSelect(p.id)
                  ? undefined
                  : t("settings.subscription.pricing.downgradeViaCancel")
              }
              currentBadge={t("settings.subscription.pricing.currentPlanBadge")}
              ctaLabel={
                isSubscriber
                  ? t("settings.subscription.pricing.switchPlan")
                  : t("settings.subscription.pricing.choosePlan")
              }
              popularLabel={t("settings.subscription.pricing.popular")}
            />
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
