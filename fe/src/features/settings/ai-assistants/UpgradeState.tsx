import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FEATURES } from "@/shared/lib/features";
import { Button } from "@/shared/ui/Button";
import { PricingModal } from "@/features/subscription/components/PricingModal";

export function UpgradeState({ showBadge = false, onClosed }: { showBadge?: boolean; onClosed?: () => void }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <div className="space-y-3">
      {showBadge && <span className="inline-block rounded border border-primary/30 px-2 py-1 font-mono text-xs text-primary-ink">Pro</span>}
      <p className="text-sm text-muted-foreground">
        {t("settings.aiAssistants.upgradeDescription")}
      </p>
      {FEATURES.PAYMENTS ? (
        <>
          <Button onClick={() => setOpen(true)}>
            {t("settings.aiAssistants.upgrade")}
          </Button>
          <PricingModal open={open} onOpenChange={(value) => {
            setOpen(value);
            if (!value) onClosed?.();
          }} />
        </>
      ) : (
        <p className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
          {t("settings.subscription.paymentsDisabled")}
        </p>
      )}
    </div>
  );
}

