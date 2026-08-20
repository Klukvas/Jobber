import { useTranslation } from "react-i18next";
import type { OverviewAnalytics } from "@/services/analyticsService";
import { Card, CardContent } from "@/shared/ui/Card";
import { Skeleton } from "@/shared/ui/Skeleton";
import {
  TrendingUp,
  Clock,
  Activity,
  CheckCircle,
  Briefcase,
} from "lucide-react";
import { cn } from "@/shared/lib/utils";

export function OverviewCards({
  data,
  isLoading,
}: {
  data?: OverviewAnalytics;
  isLoading: boolean;
}) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-5">
        {Array.from({ length: 5 }).map((_, i) => (
          <Card key={i}>
            <CardContent className="p-6">
              <Skeleton className="h-4 w-24 mb-2" />
              <Skeleton className="h-8 w-16" />
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (!data) return null;

  const cards = [
    {
      title: t("analytics.overview.totalApplications"),
      value: data.total_applications,
      icon: Briefcase,
      color: "text-blue-500 dark:text-blue-400",
    },
    {
      title: t("analytics.overview.activeApplications"),
      value: data.active_applications,
      icon: Activity,
      color: "text-green-500 dark:text-green-400",
    },
    {
      title: t("analytics.overview.closedApplications"),
      value: data.closed_applications,
      icon: CheckCircle,
      color: "text-gray-500 dark:text-gray-400",
    },
    {
      title: t("analytics.overview.responseRate"),
      value: `${data.response_rate}%`,
      icon: TrendingUp,
      color: "text-purple-500 dark:text-purple-400",
    },
    {
      title: t("analytics.overview.avgResponseTime"),
      value:
        data.avg_days_to_first_response > 0
          ? `${data.avg_days_to_first_response} ${t("analytics.days")}`
          : "-",
      icon: Clock,
      color: "text-orange-500 dark:text-orange-400",
    },
  ];

  return (
    <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-5">
      {cards.map((card) => {
        const Icon = card.icon;
        return (
          <Card key={card.title}>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">
                    {card.title}
                  </p>
                  <p className="text-2xl font-bold mt-1">{card.value}</p>
                </div>
                <Icon className={cn("h-8 w-8", card.color)} />
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
