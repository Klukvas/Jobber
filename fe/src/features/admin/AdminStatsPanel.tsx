import { useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Users, UserPlus, CalendarDays } from "lucide-react";
import { adminService } from "@/services/adminService";
import { useAdminScope } from "@/shared/hooks/useAdminAccess";
import { adminKeys } from "@/shared/lib/adminCapability";
import { Button } from "@/shared/ui/Button";

export function AdminStatsPanel() {
  const { t, i18n } = useTranslation();
  const scope = useAdminScope();
  const [days, setDays] = useState<7 | 30>(30);
  const [includeTest, setIncludeTest] = useState(false);
  const refreshing = useRef(false);
  const stats = useQuery({
    queryKey: [...adminKeys.stats(scope), days, includeTest],
    queryFn: ({ signal }) => adminService.stats(days, includeTest, signal),
    enabled: !!scope.actor && !scope.denied && !scope.statsUpdates,
    retry: false,
  });
  const data = stats.data;
  const refresh = async () => {
    if (
      refreshing.current ||
      stats.isFetching ||
      scope.statsUpdates ||
      scope.denied ||
      !scope.actor
    )
      return;
    refreshing.current = true;
    try {
      await stats.refetch();
    } finally {
      refreshing.current = false;
    }
  };
  const daily = data?.daily_registrations ?? [];
  const max = Math.max(1, ...daily.map((day) => day.count));
  const format = (value: string, full = false) =>
    new Intl.DateTimeFormat(i18n.language, {
      timeZone: "Europe/Kyiv",
      day: "numeric",
      month: full ? "long" : "short",
      ...(full ? { year: "numeric" as const } : {}),
    }).format(new Date(`${value}T12:00:00Z`));
  const number = new Intl.NumberFormat(i18n.language);
  const cards = [
    { label: t("admin.statsTotal"), value: data?.total_users, icon: Users },
    {
      label: t("admin.stats7"),
      value: data?.registrations_7_days,
      icon: UserPlus,
    },
    {
      label: t("admin.stats30"),
      value: data?.registrations_30_days,
      icon: CalendarDays,
    },
  ];
  if (scope.denied || !scope.actor) return null;
  return (
    <section aria-label={t("admin.statistics")} className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-lg font-semibold">{t("admin.statistics")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("admin.statsScope")}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <label className="flex min-h-11 cursor-pointer items-center gap-2 rounded-md px-1 text-sm">
            <input
              type="checkbox"
              checked={includeTest}
              onChange={(event) => setIncludeTest(event.target.checked)}
              className="h-4 w-4 accent-primary"
            />
            {t("admin.includeTest")}
          </label>
          <Button
            variant="outline"
            className="min-h-11"
            disabled={stats.isFetching || !!scope.statsUpdates}
            onClick={() => void refresh()}
          >
            {stats.isFetching
              ? t("admin.refreshingStats")
              : t("admin.refreshStats")}
          </Button>
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-3">
        {cards.map(({ label, value, icon: Icon }) => (
          <article key={label} className="rounded-xl border bg-card p-5">
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-sm font-medium text-muted-foreground">
                {label}
              </h3>
              <Icon
                className="h-4 w-4 text-muted-foreground"
                aria-hidden="true"
              />
            </div>
            {value === undefined && stats.isError ? (
              <p className="mt-2 text-3xl font-semibold">—</p>
            ) : value === undefined ? (
              <div
                aria-hidden="true"
                className="mt-3 h-9 w-20 animate-pulse rounded bg-muted motion-reduce:animate-none"
              />
            ) : (
              <p className="mt-2 text-3xl font-semibold tracking-tight tabular-nums">
                {number.format(value)}
              </p>
            )}
          </article>
        ))}
      </div>
      <div
        className="rounded-xl border bg-card p-4 sm:p-5"
        aria-busy={stats.isFetching || !!scope.statsUpdates}
      >
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 id="admin-registration-chart" className="font-semibold">
              {t("admin.dailyRegistrations")}
            </h3>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("admin.statsTimezone")}
            </p>
          </div>
          <div
            className="flex gap-1 rounded-lg border p-1"
            role="group"
            aria-label={t("admin.chartRange")}
          >
            {([7, 30] as const).map((range) => (
              <Button
                key={range}
                variant={days === range ? "secondary" : "ghost"}
                className="min-h-11 px-3"
                aria-pressed={days === range}
                onClick={() => setDays(range)}
              >
                {t("admin.rangeDays", { days: range })}
              </Button>
            ))}
          </div>
        </div>
        {stats.isError ? (
          <div className="flex min-h-48 flex-col items-center justify-center gap-3 text-center">
            <p role="alert" className="text-sm">
              {t("admin.statsError")}
            </p>
            <Button
              variant="outline"
              disabled={stats.isFetching || !!scope.statsUpdates}
              onClick={() => void refresh()}
            >
              {t("admin.retryStats")}
            </Button>
          </div>
        ) : !data ? (
          <div
            role="status"
            aria-label={t("admin.loadingStats")}
            className="mt-5 h-48 animate-pulse rounded bg-muted motion-reduce:animate-none"
          />
        ) : (
          <>
            <div
              className="mt-5 flex items-center justify-between text-xs tabular-nums text-muted-foreground"
              aria-hidden="true"
            >
              <span>{number.format(max)}</span>
              <span>
                {stats.isFetching
                  ? t("admin.refreshingStats")
                  : t("admin.registrations")}
              </span>
            </div>
            <svg
              preserveAspectRatio="none"
              viewBox="0 0 720 180"
              role="img"
              aria-labelledby="admin-registration-chart admin-registration-description"
              className="mt-2 h-44 w-full overflow-visible"
            >
              <desc id="admin-registration-description">
                {t("admin.chartDescription", {
                  days,
                  count: daily.reduce((sum, day) => sum + day.count, 0),
                })}
              </desc>
              {[0, 0.5, 1].map((line) => (
                <line
                  key={line}
                  x1="0"
                  x2="720"
                  y1={4 + line * 170}
                  y2={4 + line * 170}
                  stroke="currentColor"
                  className="text-border"
                  strokeWidth="1"
                />
              ))}
              {daily.map((day, index) => {
                const width = 720 / days;
                const height = (day.count / max) * 160;
                return (
                  <rect
                    key={day.date}
                    tabIndex={day.count ? 0 : undefined}
                    role="img"
                    aria-label={`${format(day.date, true)}: ${number.format(day.count)}`}
                    x={index * width + width * 0.18}
                    y={174 - height}
                    width={width * 0.64}
                    height={height}
                    rx="2"
                    fill="currentColor"
                    className="text-primary outline-offset-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
                  >
                    <title>
                      {format(day.date, true)}: {number.format(day.count)}
                    </title>
                  </rect>
                );
              })}
            </svg>
            <div
              aria-hidden="true"
              className="mt-1 flex justify-between gap-2 text-xs tabular-nums text-muted-foreground"
            >
              {daily
                .filter(
                  (_day, index) =>
                    index === 0 ||
                    index === Math.floor((days - 1) / 2) ||
                    index === days - 1,
                )
                .map((day) => (
                  <span key={day.date}>{format(day.date)}</span>
                ))}
            </div>
            {daily.every((day) => !day.count) && (
              <p className="mt-3 text-sm text-muted-foreground">
                {t("admin.noRegistrations")}
              </p>
            )}
            <div className="mt-4 flex flex-wrap items-center justify-between gap-2 border-t pt-3 text-xs text-muted-foreground">
              <p>
                {includeTest
                  ? t("admin.testsIncluded")
                  : t("admin.testsExcluded")}
              </p>
              <p>
                {t("admin.statsAsOf", {
                  time: new Intl.DateTimeFormat(i18n.language, {
                    timeZone: "Europe/Kyiv",
                    year: "numeric",
                    month: "short",
                    day: "numeric",
                    hour: "2-digit",
                    minute: "2-digit",
                  }).format(new Date(data.as_of)),
                })}
              </p>
            </div>
            <details className="mt-3 text-sm">
              <summary className="min-h-11 cursor-pointer content-center rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                {t("admin.viewDailyData")}
              </summary>
              <table className="mt-2 w-full text-left tabular-nums">
                <caption className="sr-only">
                  {t("admin.dailyRegistrations")}
                </caption>
                <thead>
                  <tr className="border-b">
                    <th scope="col" className="py-2 font-medium">
                      {t("admin.registrationDate")}
                    </th>
                    <th scope="col" className="py-2 text-right font-medium">
                      {t("admin.registrations")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {daily.map((day) => (
                    <tr key={day.date} className="border-b last:border-0">
                      <th scope="row" className="py-2 font-normal">
                        {format(day.date, true)}
                      </th>
                      <td className="py-2 text-right">
                        {number.format(day.count)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </details>
          </>
        )}
      </div>
    </section>
  );
}
