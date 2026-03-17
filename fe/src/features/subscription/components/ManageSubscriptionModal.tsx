import { useState } from "react";
import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, ArrowUpCircle, ArrowDownCircle } from "lucide-react";
import { Dialog } from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
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

const PLAN_PRICE: Record<SubscriptionPlan, string> = {
  free: "$0",
  pro: "$7/mo",
  enterprise: "$19/mo",
};

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
  const isLoading = changePlanMutation.isPending || cancelMutation.isPending;

  if (!subscription || plan === "free") return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <div className="w-full max-w-md space-y-5 bg-background rounded-lg border p-6 shadow-lg">
        <div>
          <h2 className="text-lg font-semibold">
            {t("settings.subscription.manage.title")}
          </h2>
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
              {PLAN_PRICE[plan]}
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
                  {PLAN_PRICE[targetPlan]}
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

        {/* Cancel section */}
        {!isCancelled && (
          <div className="border-t pt-4">
            {!confirmCancel ? (
              <button
                className="text-sm text-destructive hover:underline disabled:opacity-50"
                onClick={() => setConfirmCancel(true)}
                disabled={isLoading}
              >
                {t("settings.subscription.manage.cancelSubscription")}
              </button>
            ) : (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 space-y-3">
                <div className="flex gap-2">
                  <AlertTriangle className="h-4 w-4 text-destructive mt-0.5 shrink-0" />
                  <p className="text-sm text-destructive">
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
