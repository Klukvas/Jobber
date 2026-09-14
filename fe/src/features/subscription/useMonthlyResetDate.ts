import { useMemo } from "react";
import { useTranslation } from "react-i18next";

/**
 * Localized date on which the per-month allowances come back — the first of
 * next month, matching the calendar-month window the backend counts over
 * (CountUserAIRequestsThisMonth / CountUserJobParsesThisMonth).
 *
 * Only those two are monthly. Jobs, resumes, resume builders and cover letters
 * are lifetime totals and never reset, so this date must not be shown for them.
 *
 * Shared so the upgrade banner, the quota modal and any other limit copy all
 * quote the same day.
 *
 * Computed and rendered in UTC, because that is the clock the window is
 * measured against: the backend's `date_trunc('month', NOW())` runs in the
 * database's own time zone. Read in local time, the two disagree for anyone far
 * enough east or west of it for a day or so either side of the boundary — a
 * visitor at UTC+13 who is already into the first of the month is told their
 * allowance comes back on the *following* first, while the server has not
 * reset it yet either.
 */
export function useMonthlyResetDate(): string {
  const { i18n } = useTranslation();

  return useMemo(() => {
    const now = new Date();
    // `Date.UTC` rolls December over into January of the next year on its own.
    const firstOfNextMonth = new Date(
      Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1),
    );
    return firstOfNextMonth.toLocaleDateString(i18n.language, {
      year: "numeric",
      month: "long",
      day: "numeric",
      timeZone: "UTC",
    });
  }, [i18n.language]);
}
