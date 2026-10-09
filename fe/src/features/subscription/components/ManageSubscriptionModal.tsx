import { useState } from "react";
import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  ArrowUpCircle,
  ArrowDownCircle,
  ExternalLink,
} from "lucide-react";
import { Dialog, DialogTitle } from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { safeBillingUrl } from "@/features/subscription/safeHttpsUrl";
import { subscriptionService } from "@/services/subscriptionService";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import type { SubscriptionPlan } from "@/shared/types/api";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * The i18n key holding a plan's price, so this modal quotes the same copy the
 * pricing modal does.
 *
 * It used to be a literal table — "$7/mo", "$19/mo" — which was a second copy
 * of the price *and* English-only: RU and UK translate the same strings as
 * "$7/мес" and "$7/міс", so a Russian-speaking subscriber read "/mo" here and
 * "/мес" one screen away, and every price change had two places to land in.
 */
function planPriceKey(plan: SubscriptionPlan): string {
  return `settings.subscription.pricing.${plan}Price`;
}

export function ManageSubscriptionModal({ open, onOpenChange }: Props) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  const queryClient = useQueryClient();
  const { plan, subscription } = useSubscription();
  const [confirmCancel, setConfirmCancel] = useState(false);

  const changePlanMutation = useMutation({
    mutationFn: (newPlan: SubscriptionPlan) =>
      subscriptionService.changePlan(newPlan),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["subscription"] });
      showSuccessNotification(t("settings.subscription.manage.planChanged"));
      onOpenChange(false);
    },
    onError: () => {
      showErrorNotification(t("settings.subscription.manage.changePlanError"));
    },
  });

  // Same-tab navigation, never window.open: the URL only exists after an async
  // round-trip, by which point a new window is no longer tied to the user's click and
  // browsers block it.
  const portalMutation = useMutation({
    mutationFn: subscriptionService.createPortalSession,
    onSuccess: ({ url }) => {
      // The URL is backend data: treat one that is not a billing-provider https link as a
      // failed request rather than navigating to it.
      const safeUrl = safeBillingUrl(url);
      if (!safeUrl) {
        showErrorNotification(t("settings.subscription.manage.portalError"));
        return;
      }
      window.location.assign(safeUrl);
    },
    onError: () => {
      showErrorNotification(t("settings.subscription.manage.portalError"));
    },
  });

  const cancelMutation = useMutation({
    mutationFn: subscriptionService.cancelSubscription,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["subscription"] });
      showSuccessNotification(
        t("settings.subscription.manage.cancelScheduled"),
      );
      setConfirmCancel(false);
      onOpenChange(false);
    },
    onError: () => {
      showErrorNotification(t("settings.subscription.manage.cancelError"));
    },
  });

  const targetPlan: SubscriptionPlan =
    plan === "enterprise" ? "pro" : "enterprise";
  const isDowngrade = plan === "enterprise";
  const isCancelled = !!subscription?.cancel_at;
  const isLoading =
    changePlanMutation.isPending ||
    cancelMutation.isPending ||
    portalMutation.isPending;

  if (!subscription || plan === "free") return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <div className="w-full max-w-md space-y-5 bg-background rounded-lg border p-6 shadow-lg">
        <div>
          {/* The shared heading, not a bare `h2`: it is what tells the dialog
              its own name, and this was the one modal in the app that had no
              accessible name at all. */}
          <DialogTitle>{t("settings.subscription.manage.title")}</DialogTitle>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t("settings.subscription.manage.description")}
          </p>
        </div>

        {/* Current plan */}
        <div className="rounded-lg border bg-muted/40 px-4 py-3">
          <p className="text-xs text-muted-foreground uppercase tracking-wide mb-0.5">
            {t("settings.subscription.currentPlan")}
          </p>
          <div className="flex items-baseline gap-2">
            <span className="font-semibold text-base">
              {t(`settings.subscription.${plan}Plan`)}
            </span>
            <span className="text-sm text-muted-foreground">
              {t(planPriceKey(plan))}
            </span>
          </div>
          {subscription?.current_period_end && !isCancelled && (
            <p className="text-xs text-muted-foreground mt-1">
              {t("settings.subscription.renewsOn", {
                date: format(new Date(subscription.current_period_end), "PP", {
                  locale: dateLocale,
                }),
              })}
            </p>
          )}
          {isCancelled && subscription?.cancel_at && (
            <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
              {t("settings.subscription.cancelledOn", {
                date: format(new Date(subscription.cancel_at), "PP", {
                  locale: dateLocale,
                }),
              })}
            </p>
          )}
        </div>

        {/* Change plan */}
        <div className="rounded-lg border px-4 py-3 space-y-3">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="flex items-center gap-1.5">
                {isDowngrade ? (
                  <ArrowDownCircle className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ArrowUpCircle className="h-4 w-4 text-blue-500" />
                )}
                <span className="font-medium text-sm">
                  {t(`settings.subscription.${targetPlan}Plan`)}
                </span>
                <span className="text-sm text-muted-foreground">
                  {t(planPriceKey(targetPlan))}
                </span>
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">
                {t("settings.subscription.manage.proratedNote")}
              </p>
            </div>
            <Button
              size="sm"
              variant={isDowngrade ? "outline" : "default"}
              onClick={() => changePlanMutation.mutate(targetPlan)}
              disabled={isLoading}
            >
              {changePlanMutation.isPending
                ? t("common.loading")
                : isDowngrade
                  ? t("settings.subscription.manage.switchToPro")
                  : t("settings.subscription.manage.switchToEnterprise")}
            </Button>
          </div>
        </div>

        {/* Billing portal — invoices, receipts and the payment method live on
            the provider's side, so this is the only route to them. */}
        <div className="rounded-lg border px-4 py-3">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="flex items-center gap-1.5">
                <ExternalLink className="h-4 w-4 text-muted-foreground" />
                <span className="font-medium text-sm">
                  {t("settings.subscription.manage.billingPortal")}
                </span>
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">
                {t("settings.subscription.manage.billingPortalDescription")}
              </p>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={() => portalMutation.mutate()}
              disabled={isLoading}
            >
              {portalMutation.isPending
                ? t("common.loading")
                : t("settings.subscription.manage.openBillingPortal")}
            </Button>
          </div>
        </div>

        {/* Cancel section */}
        {!isCancelled && (
          <div className="border-t pt-4">
            {!confirmCancel ? (
              <button
                type="button"
                className="text-sm text-destructive hover:underline disabled:opacity-50"
                onClick={() => setConfirmCancel(true)}
                disabled={isLoading}
              >
                {t("settings.subscription.manage.cancelSubscription")}
              </button>
            ) : (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 space-y-3">
                <div className="flex gap-2">
                  <AlertTriangle
                    aria-hidden="true"
                    className="h-4 w-4 text-destructive mt-0.5 shrink-0"
                  />
                  {/* Announced when it appears — the consequence of cancelling
                      must reach a screen reader, not only the sighted user. */}
                  <p role="alert" className="text-sm text-destructive">
                    {t("settings.subscription.manage.cancelConfirmText")}
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => cancelMutation.mutate()}
                    disabled={isLoading}
                  >
                    {cancelMutation.isPending
                      ? t("common.loading")
                      : t("settings.subscription.manage.confirmCancel")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setConfirmCancel(false)}
                    disabled={isLoading}
                  >
                    {t("common.cancel")}
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </Dialog>
  );
}
