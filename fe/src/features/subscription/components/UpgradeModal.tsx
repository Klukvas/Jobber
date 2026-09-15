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
    isCheckoutPopupOpen,
    errorMessageKey,
  } = usePlanSelection({
    onPlanChanged: () => onOpenChange(false),
  });

  // AI requests are counted per calendar month, so the quota genuinely returns.
  const resetDate = useMonthlyResetDate();

  if (!FEATURES.PAYMENTS) return null;

  // A free user gets the provider's checkout popup drawn over this page —
  // nothing navigates, and no URL is involved. The modal stays open behind it
  // on purpose: if starting the checkout failed, this is where the user sees
  // why. A subscriber upgrades in place and the modal closes on success.
  const handleUpgrade = () => {
    if (nextPlan) {
      selectPlan(nextPlan);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      // The payment popup is appended outside this dialog: while it is up, the
      // keyboard has to reach it, and Escape must not close the page behind a
      // payment in flight.
      hasExternalOverlay={isCheckoutPopupOpen}
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
