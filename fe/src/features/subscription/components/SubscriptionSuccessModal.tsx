import { useTranslation } from "react-i18next";
import { Sparkles } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/Dialog";
import type { SubscriptionPlan } from "@/shared/types/api";

interface SubscriptionSuccessModalProps {
  plan: SubscriptionPlan | null;
  onClose: () => void;
}

export function SubscriptionSuccessModal({
  plan,
  onClose,
}: SubscriptionSuccessModalProps) {
  const { t } = useTranslation();

  const planLabel =
    plan === "pro"
      ? t("settings.subscription.proPlan")
      : plan === "enterprise"
        ? t("settings.subscription.enterprisePlan")
        : "";

  return (
    <Dialog open={!!plan} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        onClose={onClose}
        className="max-w-sm text-center sm:max-w-md"
      >
        <DialogHeader>
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary">
            <Sparkles className="h-8 w-8 text-primary-foreground" />
          </div>
          <DialogTitle className="text-center text-2xl font-bold">
            {t("settings.subscription.upgradeSuccess.title")}
          </DialogTitle>
        </DialogHeader>

        <p className="mt-2 text-muted-foreground">
          {t("settings.subscription.upgradeSuccess.description", {
            plan: planLabel,
          })}
        </p>

        <button
          onClick={onClose}
          className="mt-6 w-full rounded-md bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground shadow-sm transition-all hover:bg-primary/90 hover:shadow-md"
        >
          {t("settings.subscription.upgradeSuccess.cta")}
        </button>
      </DialogContent>
    </Dialog>
  );
}
