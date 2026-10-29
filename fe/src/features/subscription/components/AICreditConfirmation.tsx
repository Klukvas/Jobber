import { useLayoutEffect, useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { subscriptionService } from "@/services/subscriptionService";
import { aiConsentStore } from "@/shared/lib/aiCredits";
import { Button } from "@/shared/ui/Button";
import { Dialog, DialogContent, DialogTitle, DialogFooter } from "@/shared/ui/Dialog";

export function AICreditConfirmation({ originKey }: { readonly originKey?: string }) {
  const { t } = useTranslation();
  useLayoutEffect(() => () => { aiConsentStore.getSnapshot()?.finish(false); }, [originKey]);
  const pending = useSyncExternalStore(aiConsentStore.subscribe, aiConsentStore.getSnapshot);
  const balance = useQuery({
    queryKey: ["subscription"], queryFn: subscriptionService.getSubscription,
    enabled: !!pending, staleTime: 0, retry: false,
  });
  if (!pending) return null;
  const cost = balance.data?.credit_costs?.[pending.operation];
  const remaining = balance.data?.credits?.remaining;
  const known = !balance.isFetching && !balance.isError && typeof cost === "number" && Number.isSafeInteger(cost) && cost > 0 && typeof remaining === "number" && Number.isSafeInteger(remaining) && remaining >= 0;
  const enough = known && remaining >= cost;
  return createPortal(
    <div className="relative z-[100]">
      <Dialog open onOpenChange={() => pending.finish(false)}>
        <DialogContent onClose={() => pending.finish(false)}>
          <DialogTitle>{t("credits.confirmTitle")}</DialogTitle>
          <div className="my-5 space-y-3 text-sm">
            <p className="font-medium">{t(`credits.operations.${pending.operation}`)}</p>
            {known ? <p>{t("credits.cost", { cost, remaining })}</p> : <p role="status">{t(balance.isFetching ? "common.loading" : "credits.balanceUnavailable")}</p>}
            {known && !enough && <p role="alert" className="text-destructive">{t("credits.exhausted")}</p>}
            <p>{t("credits.chargePolicy")}</p>
            <p className="text-muted-foreground">{t("credits.allowance")}</p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => pending.finish(false)}>{t("common.cancel")}</Button>
            <Button disabled={!enough} onClick={() => pending.finish(true)}>{t("credits.confirm", { cost })}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>, document.body,
  );
}
