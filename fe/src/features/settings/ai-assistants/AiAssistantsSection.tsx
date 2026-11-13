import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { apiTokensService } from "@/services/apiTokensService";
import { ApiError } from "@/services/api";
import { useAuthStore } from "@/stores/authStore";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { FEATURES } from "@/shared/lib/features";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { Skeleton } from "@/shared/ui/Skeleton";
import { PricingModal } from "@/features/subscription/components/PricingModal";
import { CreateTokenForm } from "./CreateTokenForm";
import { TokenList } from "./TokenList";
import { ActivityList } from "./ActivityList";

function UpgradeState({ showBadge = false }: { showBadge?: boolean }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <div className="space-y-3">
      {showBadge && <span className="inline-block rounded border border-primary/30 px-2 py-1 font-mono text-xs text-primary">Pro</span>}
      <p className="text-sm text-muted-foreground">
        {t("settings.aiAssistants.upgradeDescription")}
      </p>
      {FEATURES.PAYMENTS ? (
        <>
          <Button onClick={() => setOpen(true)}>
            {t("settings.aiAssistants.upgrade")}
          </Button>
          <PricingModal open={open} onOpenChange={setOpen} />
        </>
      ) : (
        <p className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
          {t("settings.subscription.paymentsDisabled")}
        </p>
      )}
    </div>
  );
}

function isPaidError(error: unknown): boolean {
  return error instanceof ApiError && error.code === "PAID_FEATURE";
}

function PaidAssistants({ userId }: { userId: string }) {
  const queryClient = useQueryClient();
  const [denied, setDenied] = useState(false);
  const key = ["api-tokens", userId];
  const tokens = useQuery({
    queryKey: key,
    queryFn: ({ signal }) => apiTokensService.list({ signal }),
    enabled: !denied,
    retry: false,
    refetchInterval: 60_000,
  });
  const activity = useQuery({
    queryKey: [...key, "activity"],
    queryFn: ({ signal }) => apiTokensService.activity({ signal }),
    enabled: !denied,
    retry: false,
    refetchInterval: 60_000,
  });
  if (denied || isPaidError(tokens.error) || isPaidError(activity.error)) {
    return <UpgradeState showBadge />;
  }
  const refresh = () => {
    return queryClient.invalidateQueries({ queryKey: key });
  };
  return (
    <div className="min-w-0 space-y-6">
      <CreateTokenForm onCreated={refresh} onPaidFeature={() => setDenied(true)} />
      <TokenList query={tokens} onRevoked={refresh} onPaidFeature={() => setDenied(true)} />
      <ActivityList query={activity} />
    </div>
  );
}

export function AiAssistantsSection() {
  const { t } = useTranslation();
  const { isPro, isLoading } = useSubscription();
  const userId = useAuthStore((state) => state.user?.id);
  return (
    <Card className="min-w-0">
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2">
          {t("settings.aiAssistants.title")}
          {!isLoading && !isPro && (
            <span className="rounded border border-primary/30 px-2 py-1 font-mono text-xs text-primary">Pro</span>
          )}
        </CardTitle>
        <CardDescription>{t("settings.aiAssistants.description")}</CardDescription>
      </CardHeader>
      <CardContent className="min-w-0">
        {isLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : isPro && userId ? (
          <PaidAssistants key={userId} userId={userId} />
        ) : <UpgradeState />}
      </CardContent>
    </Card>
  );
}
