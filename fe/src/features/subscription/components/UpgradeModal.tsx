import { useMemo } from "react";
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
import { usePaddleCheckout } from "@/features/subscription/usePaddleCheckout";
import { FEATURES } from "@/shared/lib/features";

interface UpgradeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UpgradeModal({ open, onOpenChange }: UpgradeModalProps) {
  if (!FEATURES.PAYMENTS) return null;
  const { t, i18n } = useTranslation();
  const { nextPlan } = useSubscription();
  const { openCheckout, isReady } = usePaddleCheckout();

  const resetDate = useMemo(() => {
    const now = new Date();
    const firstOfNextMonth = new Date(now.getFullYear(), now.getMonth() + 1, 1);
    return firstOfNextMonth.toLocaleDateString(i18n.language, {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  }, [i18n.language]);

  const handleUpgrade = () => {
    if (nextPlan) {
      openCheckout(nextPlan);
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("settings.subscription.aiLimitTitle")}</DialogTitle>
          <DialogDescription>
            {t("settings.subscription.aiLimitMessage", { date: resetDate })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.close")}
          </Button>
          <Button onClick={handleUpgrade} disabled={!isReady || !nextPlan}>
            {t("settings.subscription.upgradeForMore")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
