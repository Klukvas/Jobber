/**
 * Query keys for the cover letter caches.
 *
 * The editor, the letters list and the auto-save queue all reach the same two
 * entries, and they used to spell them by hand in three places. One factory
 * keeps them in step — and makes the rule visible: the list key must never be a
 * prefix of a detail key, or invalidating the list would also refetch the
 * letter the editor is showing, out from under the cursor.
 *
 * `coverLetterQueryKeys.test.ts` pins both properties down against a real
 * QueryClient, because which key matches which is React Query's rule, not ours.
 */
export const coverLetterQueryKeys = {
  /** Every cover letter of the current user — the list behind the editor. */
  list: () => ["cover-letters"] as const,
  /** One cover letter with its full content. */
  detail: (id: string) => ["cover-letter", id] as const,
} as const;
