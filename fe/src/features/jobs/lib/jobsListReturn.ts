import { useLocation } from "react-router-dom";

/**
 * Navigation state a job card hands to the detail page so its Back button can
 * return to the list *as it was* — page, search, sort and filters included —
 * instead of a fresh, unfiltered `/app/jobs`.
 */
export interface JobsListReturnState {
  from: string;
}

/** The current list URL, to attach to a card's navigation. */
export function useJobsListReturnState(): JobsListReturnState {
  const { pathname, search } = useLocation();
  return { from: `${pathname}${search}` };
}

/**
 * Where Back goes: the list URL the card was opened from, or the bare list
 * when the card was opened directly (a shared link, a new tab, a refresh).
 *
 * Only an in-app jobs URL is accepted. Navigation state is as user-supplied
 * as the address bar, and this value is fed straight to `navigate`.
 */
export function jobsListReturnPath(state: unknown): string {
  const from = (state as JobsListReturnState | null)?.from;
  return typeof from === "string" && /^\/app\/jobs(\?|$)/.test(from)
    ? from
    : "/app/jobs";
}
