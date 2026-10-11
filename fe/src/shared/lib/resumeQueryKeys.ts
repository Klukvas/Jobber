/**
 * Query keys for the builder-resume caches.
 *
 * The list was spelled two ways: `["resume-builders"]` on the resumes page and
 * `["resume-builder"]` in the job screens. Create / duplicate / delete / rename
 * invalidate the plural one, so the job screens kept serving a list that no
 * longer matched the server. Worse, the singular spelling is a *prefix* of every
 * `["resume-builder", id]` detail entry, so invalidating that list would also
 * sweep every cached resume.
 *
 * One factory, one spelling: the list key must never be a prefix of a detail
 * key. `__tests__/resumeQueryKeys.test.tsx` pins that down against a real
 * QueryClient — which key matches which is React Query's rule, not ours.
 */
export const resumeQueryKeys = {
  /** Every builder resume of the current user — the list behind the pickers. */
  builderList: () => ["resume-builders"] as const,
  /** One builder resume with its full content. */
  builderDetail: (id: string) => ["resume-builder", id] as const,
} as const;
