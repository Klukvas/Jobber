import { useTranslation } from "react-i18next";
import type { FunnelAnalytics } from "@/services/analyticsService";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/shared/ui/Card";
import { Skeleton } from "@/shared/ui/Skeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { TrendingUp, ArrowRight } from "lucide-react";

export function FunnelVisualization({
  data,
  isLoading,
}: {
  data?: FunnelAnalytics;
  isLoading: boolean;
}) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            {t("analytics.funnel.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4">
                <Skeleton className="h-12 flex-1" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!data?.stages || data.stages.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            {t("analytics.funnel.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<TrendingUp className="h-12 w-12" />}
            title={t("analytics.funnel.noData")}
            description={t("analytics.funnel.noDataDescription")}
          />
        </CardContent>
      </Card>
    );
  }

  const maxCount = Math.max(...data.stages.map((s) => s.count), 1);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5" />
          {t("analytics.funnel.title")}
        </CardTitle>
        <CardDescription>{t("analytics.funnel.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {data.stages.map((stage, index) => {
            const widthPercent = (stage.count / maxCount) * 100;
            return (
              <div key={stage.stage_name} className="space-y-1">
                <div className="flex flex-col gap-0.5 text-sm sm:flex-row sm:items-center sm:justify-between">
                  <span className="font-medium">{stage.stage_name}</span>
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-0.5 text-muted-foreground">
                    <span>
                      {stage.count} {t("analytics.applications")}
                    </span>
                    {index > 0 && (
                      <>
                        <span className="text-green-600">
                          {stage.conversion_rate}%{" "}
                          {t("analytics.funnel.converted")}
                        </span>
                        <span className="text-red-500">
                          {stage.drop_off_rate}% {t("analytics.funnel.dropOff")}
                        </span>
                      </>
                    )}
                  </div>
                </div>
                <div className="h-8 bg-muted rounded-md overflow-hidden">
                  <div
                    className="h-full bg-primary/80 rounded-md transition-all duration-500 flex items-center justify-end pr-2"
                    style={{ width: `${Math.max(widthPercent, 5)}%` }}
                  >
                    {widthPercent > 15 && (
                      <span className="text-xs text-primary-foreground font-medium">
                        {stage.count}
                      </span>
                    )}
                  </div>
                </div>
                {index < data.stages.length - 1 && (
                  <div className="flex justify-center py-1">
                    <ArrowRight className="h-4 w-4 text-muted-foreground rotate-90" />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
