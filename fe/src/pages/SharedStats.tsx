import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { sharingService } from "@/services/sharingService";
import { ApiError } from "@/services/api";
import { OverviewCards } from "@/features/analytics/components/OverviewCards";
import { FunnelVisualization } from "@/features/analytics/components/FunnelVisualization";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent } from "@/shared/ui/Card";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ErrorState } from "@/shared/ui/ErrorState";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { BarChart3, Link2Off } from "lucide-react";

export default function SharedStats() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { token } = useParams<{ token: string }>();
  usePageMeta({ titleKey: "sharing.public.title", noindex: true });

  const shareQuery = useQuery({
    queryKey: ["public-share", token],
    queryFn: () => sharingService.getPublic(token ?? ""),
    enabled: Boolean(token),
    retry: false,
  });

  const isNotFound =
    !token ||
    (shareQuery.error instanceof ApiError && shareQuery.error.status === 404);

  const snapshot = shareQuery.data?.snapshot;

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4">
          <Link to="/" className="text-xl font-bold text-primary">
            Jobber
          </Link>
          <Button size="sm" onClick={() => navigate("/register")}>
            {t("sharing.public.ctaButton")}
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl space-y-6 px-4 py-8">
        {isNotFound ? (
          <EmptyState
            icon={<Link2Off className="h-16 w-16" />}
            title={t("sharing.public.notFoundTitle")}
            description={t("sharing.public.notFoundDescription")}
          />
        ) : shareQuery.isError ? (
          <ErrorState
            message={t("sharing.public.loadError")}
            onRetry={() => shareQuery.refetch()}
          />
        ) : (
          <>
            <div className="flex items-center gap-3">
              <BarChart3 className="h-8 w-8 text-primary" />
              <div>
                <h1 className="text-3xl font-bold">
                  {t("sharing.public.title")}
                </h1>
                {shareQuery.data && (
                  <p className="text-muted-foreground">
                    {t("sharing.public.sharedOn", {
                      date: new Date(
                        shareQuery.data.created_at,
                      ).toLocaleDateString(),
                    })}
                  </p>
                )}
              </div>
            </div>

            <OverviewCards
              data={snapshot?.overview}
              isLoading={shareQuery.isLoading}
            />
            <FunnelVisualization
              data={snapshot ? { stages: snapshot.funnel } : undefined}
              isLoading={shareQuery.isLoading}
            />
          </>
        )}

        <Card>
          <CardContent className="flex flex-col items-center gap-3 p-8 text-center">
            <h2 className="text-xl font-semibold">
              {t("sharing.public.ctaTitle")}
            </h2>
            <p className="text-muted-foreground">
              {t("sharing.public.ctaDescription")}
            </p>
            <Button onClick={() => navigate("/register")}>
              {t("sharing.public.ctaButton")}
            </Button>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
