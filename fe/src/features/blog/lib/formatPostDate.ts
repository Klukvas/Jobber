import { format, parseISO } from "date-fns";
import type { Locale } from "date-fns";

/**
 * A post's publication date, written the way its reader's language writes one.
 *
 * `PPP` is date-fns' *localized* long date: the locale supplies the order as
 * well as the words. The pattern used to be a literal `MMMM d, yyyy`, which
 * translated the month name and kept the American order — so a Russian reader
 * got "января 15, 2025" and a Ukrainian one "січня 15, 2025", a translated word
 * dropped into an English sentence. Both languages put the day first.
 *
 * One function rather than the same call in two components, so the article and
 * the card it is listed on cannot drift apart.
 */
export function formatPostDate(isoDate: string, locale: Locale): string {
  return format(parseISO(isoDate), "PPP", { locale });
}
