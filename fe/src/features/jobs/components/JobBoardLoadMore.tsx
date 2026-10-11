import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { Button } from "@/shared/ui/Button";

interface JobBoardLoadMoreProps {
  /** Cards currently on the board. */
  loaded: number;
  /** Cards the filter matches server-side; undefined before the first page. */
  total?: number;
  hasMore: boolean;
  isLoadingMore: boolean;
  /** The last "load more" failed — the cards already here are still valid. */
  failed: boolean;
  onLoadMore: () => void;
}

/**
 * The board's footer: how much of the pipeline is on screen, and the way to
 * bring in the rest.
 *
 * The board used to request a flat 500 cards and show whatever arrived, so a
 * user with more than that had cards that existed in the list view and simply
 * were not on the board — with nothing on screen saying so. Stating
 * "loaded of total" is the point of this component; the button is secondary.
 */
export function JobBoardLoadMore({
  loaded,
  total,
  hasMore,
  isLoadingMore,
  failed,
  onLoadMore,
}: JobBoardLoadMoreProps) {
  const { t } = useTranslation();

  if (total === undefined) return null;

  return (
    <div className="flex flex-wrap items-center justify-center gap-3">
      <p role="status" className="text-sm text-muted-foreground">
        {t("jobs.board.loadedCount", { loaded, total })}
      </p>
      {failed && (
        <p role="alert" className="text-sm text-destructive">
          {t("jobs.board.loadMoreError")}
        </p>
      )}
      {hasMore && (
        <Button
          variant="outline"
          size="sm"
          onClick={onLoadMore}
          disabled={isLoadingMore}
        >
          {isLoadingMore ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("jobs.board.loadingMore")}
            </>
          ) : (
            t(failed ? "common.tryAgain" : "jobs.board.loadMore")
          )}
        </Button>
      )}
    </div>
  );
}
