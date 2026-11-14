import { useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/services/api";
import { useAuthStore } from "@/stores/authStore";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { ErrorState } from "@/shared/ui/ErrorState";
import { Card, CardContent, CardHeader } from "@/shared/ui/Card";
import { ConsentLoading } from "@/features/oauth/ConsentLoading";
import { ConsentAuth } from "@/features/oauth/ConsentAuth";
import { ConsentDetails, ConsentActions, ConsentUpgrade } from "@/features/oauth/ConsentDetails";
import { useConsent } from "@/features/oauth/useConsent";

function consentErrorKey(error: unknown, fallback: string): string {
  if (!(error instanceof ApiError)) return fallback;
  if (error.status === 429) return "oauth.rateLimitError";
  return error.code === "ORIGIN_NOT_ALLOWED" ? "oauth.originError" : fallback;
}

function ConsentContent({ search, userId }: { search: string; userId: string | undefined }) {
  const { t } = useTranslation();
  const { query, status, error, decide } = useConsent(search, userId);
  if (!userId || (query.error instanceof ApiError && query.error.status === 401)) return <ConsentAuth onAuthenticated={() => { if (userId) void query.refetch(); }} />;
  // CIMD metadata may be unavailable until upgrade, so no callback is trusted yet.
  if (query.isError && query.error instanceof ApiError && query.error.status === 403 && query.error.code === "PAID_FEATURE") {
    return <ConsentUpgrade onRefresh={() => void query.refetch()} />;
  }
  if (query.isError) return <ErrorState message={t(consentErrorKey(query.error, "oauth.previewError"))} onRetry={() => void query.refetch()} />;
  if (!query.data) return <ConsentLoading />;
  if ("redirect_to" in query.data) return <p role="status">{t("oauth.redirecting")}</p>;
  return (
    <>
      <ConsentDetails preview={query.data} onRefresh={() => void query.refetch()} />
      <ConsentActions allowed={query.data.plan_allowed} pending={status === "pending"}
        error={status === "error" ? t(consentErrorKey(error, "oauth.submitError")) : undefined} onDecision={(decision) => void decide(decision)} />
    </>
  );
}

export default function OAuthAuthorizePage() {
  const { t } = useTranslation();
  const { search } = useLocation();
  const userId = useAuthStore((state) => state.isAuthenticated ? state.user?.id : undefined);
  usePageMeta({ titleKey: "oauth.title", noindex: true });
  return (
    <main className="min-h-dvh bg-background px-4 pt-[max(2rem,env(safe-area-inset-top))] pb-[max(2rem,env(safe-area-inset-bottom))] text-foreground">
      <Card className="mx-auto w-full max-w-xl">
        <CardHeader>
          <p className="mb-2 font-mono text-sm text-primary-ink">Cursopus · MCP</p>
          <h1 className="text-2xl font-semibold tracking-tight">{t("oauth.title")}</h1>
        </CardHeader>
        <CardContent>
          {window.self !== window.top ? <ErrorState message={t("oauth.frameError")} /> : <ConsentContent key={`${userId ?? "signed-out"}:${search}`} search={search} userId={userId} />}
        </CardContent>
      </Card>
    </main>
  );
}
