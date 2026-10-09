import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { usePlanSelection } from "@/features/subscription/usePlanSelection";
import { FEATURES } from "@/shared/lib/features";
import { useMonthlyResetDate } from "@/features/subscription/useMonthlyResetDate";

interface UpgradeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UpgradeModal({ open, onOpenChange }: UpgradeModalProps) {
  const { t } = useTranslation();
  const { nextPlan } = useSubscription();
  const {
    selectPlan,
    isReady,
    isPending,
    isCheckoutRedirecting,
    errorMessageKey,
  } = usePlanSelection({
    onPlanChanged: () => onOpenChange(false),
  });

  // AI requests are counted per calendar month, so the quota genuinely returns.
  const resetDate = useMonthlyResetDate();

  if (!FEATURES.PAYMENTS) return null;

  // A free user is sent to the provider's hosted checkout by a full-page
  // redirect. The modal stays open meanwhile on purpose: if starting the
  // checkout failed, this is where the user sees why. A subscriber upgrades in place and the modal closes on success.
  const handleUpgrade = () => {
    if (nextPlan) {
      selectPlan(nextPlan);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      // While the checkout request is in flight or the browser is navigating to
      // the provider's page, Escape must not close the modal: that would hide
      // the error if the start fails, and the page is about to be replaced.
      hasExternalOverlay={isCheckoutRedirecting}
    >
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("settings.subscription.aiLimitTitle")}</DialogTitle>
          <DialogDescription>
            {t("settings.subscription.aiLimitMessage", { date: resetDate })}
          </DialogDescription>
        </DialogHeader>
        {errorMessageKey && (
          <p role="alert" className="text-sm text-destructive">
            {t(errorMessageKey)}
          </p>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.close")}
          </Button>
          <Button
            onClick={handleUpgrade}
            disabled={!isReady || !nextPlan || isPending}
          >
            {t("settings.subscription.upgradeForMore")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
