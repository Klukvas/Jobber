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
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-violet-600">
            <Sparkles className="h-8 w-8 text-white" />
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
          className="mt-6 w-full rounded-md bg-gradient-to-r from-blue-600 to-violet-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-all hover:from-blue-700 hover:to-violet-700 hover:shadow-md"
        >
          {t("settings.subscription.upgradeSuccess.cta")}
        </button>
      </DialogContent>
    </Dialog>
  );
}
