import { Skeleton } from "@/shared/ui/Skeleton";

/**
 * Skeleton for list pages that show a header bar + grid of card placeholders.
 * Used by: Applications, Jobs, Companies, Resumes, StageTemplates, CoverLetters.
 */
export function ListPageSkeleton({
  cards = 6,
  columns = "md:grid-cols-2 lg:grid-cols-3",
}: {
  cards?: number;
  columns?: string;
}) {
  return (
    <div className="space-y-6" role="status" aria-label="Loading...">
      {/* Header: title + button */}
      <div className="flex items-center justify-between">
        <Skeleton className="h-9 w-48" />
        <Skeleton className="h-10 w-32" />
      </div>

      {/* Sort/filter bar */}
      <div className="flex items-center gap-2">
        <Skeleton className="h-5 w-16" />
        <Skeleton className="h-8 w-24" />
        <Skeleton className="h-8 w-24" />
        <Skeleton className="h-8 w-24" />
      </div>

      {/* Card grid */}
      <div className={`grid gap-4 ${columns}`}>
        {Array.from({ length: cards }).map((_, i) => (
          <div key={i} className="rounded-lg border bg-card p-4 space-y-3">
            <div className="flex items-start justify-between">
              <Skeleton className="h-5 w-3/4" />
              <Skeleton className="h-5 w-5 rounded-full" />
            </div>
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-4 w-2/3" />
            <div className="flex gap-2 pt-2">
              <Skeleton className="h-6 w-20 rounded-full" />
            </div>
            <div className="space-y-2 border-t pt-3">
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-4/5" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * Skeleton for detail pages with a header + content blocks.
 * Used by: ApplicationDetail, JobDetail.
 */
export function DetailPageSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-label="Loading...">
      {/* Back button + title */}
      <div className="space-y-2">
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-4 w-1/4" />
      </div>

      {/* Main content blocks */}
      <div className="space-y-4">
        <div className="rounded-lg border bg-card p-6 space-y-4">
          <Skeleton className="h-6 w-1/4" />
          <Skeleton className="h-32 w-full" />
        </div>
        <div className="rounded-lg border bg-card p-6 space-y-4">
          <Skeleton className="h-6 w-1/3" />
          <Skeleton className="h-24 w-full" />
        </div>
        <div className="rounded-lg border bg-card p-6 space-y-4">
          <Skeleton className="h-6 w-1/4" />
          <Skeleton className="h-24 w-full" />
        </div>
      </div>
    </div>
  );
}

/**
 * Skeleton for editor pages with a sidebar + main editing area.
 * Used by: ResumeBuilderEditor, CoverLetterEditor.
 */
export function EditorPageSkeleton() {
  return (
    <div
      className="flex h-[calc(100vh-4rem)] gap-4"
      role="status"
      aria-label="Loading..."
    >
      {/* Sidebar */}
      <div className="hidden md:flex w-64 shrink-0 flex-col space-y-4 rounded-lg border bg-card p-4">
        <Skeleton className="h-6 w-3/4" />
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-8 w-full" />
          ))}
        </div>
      </div>

      {/* Main content area */}
      <div className="flex-1 rounded-lg border bg-card p-6 space-y-6">
        <Skeleton className="h-8 w-1/2" />
        <div className="space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      </div>
    </div>
  );
}

/**
 * Skeleton for the cover letter list page with thumbnail-style cards.
 */
export function CoverLetterListSkeleton({ cards = 6 }: { cards?: number }) {
  return (
    <div className="space-y-6" role="status" aria-label="Loading...">
      {/* Header: title + button */}
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-44" />
        <Skeleton className="h-10 w-36" />
      </div>

      {/* Card grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: cards }).map((_, i) => (
          <div key={i} className="rounded-lg border bg-card p-4 space-y-3">
            {/* Thumbnail area */}
            <Skeleton className="h-28 w-full rounded-md" />
            {/* Title + meta */}
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * Skeleton for the stage templates list (vertical list of cards, not a grid).
 */
export function StageTemplateListSkeleton({ count = 5 }: { count?: number }) {
  return (
    <div className="space-y-6" role="status" aria-label="Loading...">
      {/* Header: title + description + button */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <Skeleton className="h-9 w-48" />
          <Skeleton className="h-4 w-64" />
        </div>
        <Skeleton className="h-10 w-28" />
      </div>

      {/* Recommended stages card */}
      <div className="rounded-lg border bg-card p-6 space-y-4">
        <div className="flex items-center gap-2">
          <Skeleton className="h-5 w-5 rounded-full" />
          <Skeleton className="h-5 w-40" />
        </div>
        <Skeleton className="h-4 w-72" />
        <div className="flex flex-wrap gap-2">
          {Array.from({ length: 7 }).map((_, i) => (
            <Skeleton key={i} className="h-8 w-28" />
          ))}
        </div>
      </div>

      {/* Stage template rows */}
      <div className="space-y-3">
        {Array.from({ length: count }).map((_, i) => (
          <div key={i} className="rounded-lg border bg-card p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Skeleton className="h-8 w-8 rounded-full" />
                <Skeleton className="h-5 w-32" />
              </div>
              <div className="flex gap-2">
                <Skeleton className="h-8 w-8" />
                <Skeleton className="h-8 w-8" />
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * Skeleton for the Settings page (stack of settings cards).
 */
export function SettingsPageSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-label="Loading...">
      <Skeleton className="h-9 w-32" />

      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="rounded-lg border bg-card">
          <div className="p-6 space-y-2">
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-4 w-64" />
          </div>
          <div className="px-6 pb-6">
            <div className="flex gap-4">
              <Skeleton className="h-10 w-24" />
              <Skeleton className="h-10 w-24" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
