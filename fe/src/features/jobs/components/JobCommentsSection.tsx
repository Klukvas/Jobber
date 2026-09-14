import { useState } from "react";
import { useTranslation } from "react-i18next";
import { formatDistanceToNow } from "date-fns";
import { Check, MessageSquarePlus, Pencil, Trash2, X } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { Textarea } from "@/shared/ui/Textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/Dialog";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import type { CommentDTO } from "@/shared/types/api";

interface JobCommentsSectionProps {
  readonly comments: readonly CommentDTO[];
  readonly newComment: string;
  readonly onChangeComment: (value: string) => void;
  readonly onAddComment: (e: React.FormEvent) => void;
  readonly isAdding: boolean;
  /**
   * Saves an edit. Must resolve only once the server has accepted it and
   * reject otherwise — this component closes the editor on the resolution and
   * keeps the draft on the rejection, and it has no other way to tell the two
   * apart. Wire it to a mutation's `mutateAsync`, never `mutate`.
   */
  readonly onUpdateComment: (
    commentId: string,
    content: string,
  ) => Promise<unknown>;
  readonly isUpdating: boolean;
  /** Deletes a comment. Same contract as `onUpdateComment`. */
  readonly onDeleteComment: (commentId: string) => Promise<unknown>;
  readonly isDeleting: boolean;
}

/**
 * Shared shape of the per-comment edit and delete controls.
 *
 * A 16px glyph in `p-1.5` is a 28x28 target — under the 44px WCAG 2.5.5 and the
 * Apple HIG both ask for, on two controls that sit a few pixels apart and one of
 * which deletes something for good. 44x44 on phones, back to the original 28x28
 * from `sm` up so pointer-driven layouts keep their density. The glyph itself is
 * unchanged and centred in the larger box, so nothing about it moves or grows.
 */
const COMMENT_ACTION_BUTTON =
  "inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-md " +
  "transition-colors focus-visible:outline-none focus-visible:ring-2 " +
  "focus-visible:ring-ring sm:h-7 sm:w-7";

/** First line of a comment, shortened — used to name it in the delete confirm. */
function commentPreview(content: string): string {
  const firstLine = content.split("\n")[0].trim();
  return firstLine.length > 60 ? `${firstLine.slice(0, 60)}…` : firstLine;
}

export function JobCommentsSection({
  comments,
  newComment,
  onChangeComment,
  onAddComment,
  isAdding,
  onUpdateComment,
  isUpdating,
  onDeleteComment,
  isDeleting,
}: JobCommentsSectionProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [pendingDelete, setPendingDelete] = useState<CommentDTO | null>(null);

  const startEditing = (comment: CommentDTO) => {
    setEditingId(comment.id);
    setDraft(comment.content);
  };

  const cancelEditing = () => {
    setEditingId(null);
    setDraft("");
  };

  /**
   * Saves the edit, and only then closes the editor.
   *
   * Closing first — which is what it used to do — threw the draft away the
   * instant the request left, so a rejected PATCH left an error toast, the
   * original text back on screen and whatever had been typed gone for good.
   * The editor now stays open on failure with the draft intact, so the obvious
   * thing to do next (press Save again) is possible.
   */
  const saveEdit = async (comment: CommentDTO) => {
    const trimmed = draft.trim();
    if (!trimmed || trimmed === comment.content) {
      cancelEditing();
      return;
    }
    try {
      await onUpdateComment(comment.id, trimmed);
      cancelEditing();
    } catch {
      // The caller has already told the customer what went wrong; all this has
      // to do is not destroy their text.
    }
  };

  /** Same rule for the delete confirm: it closes on success, not on send. */
  const confirmDelete = async () => {
    if (!pendingDelete) return;
    try {
      await onDeleteComment(pendingDelete.id);
      setPendingDelete(null);
    } catch {
      // Leave the dialog up so the action can be retried or abandoned.
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">{t("jobs.comments")}</CardTitle>
      </CardHeader>
      <CardContent>
        {comments.length > 0 && (
          <div className="space-y-3 mb-4">
            {comments.map((comment) => {
              const isEditing = editingId === comment.id;
              const wasEdited =
                !!comment.updated_at && comment.updated_at !== comment.created_at;

              return (
                <div
                  key={comment.id}
                  className="rounded-lg border bg-muted/50 p-3"
                >
                  {isEditing ? (
                    <div className="space-y-2">
                      <Textarea
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        rows={3}
                        autoFocus
                        aria-label={t("jobs.editComment")}
                      />
                      <div className="flex flex-wrap gap-2">
                        <Button
                          size="sm"
                          onClick={() => saveEdit(comment)}
                          disabled={!draft.trim() || isUpdating}
                        >
                          <Check className="h-4 w-4 mr-1" />
                          {t("common.save")}
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={cancelEditing}
                          disabled={isUpdating}
                        >
                          <X className="h-4 w-4 mr-1" />
                          {t("common.cancel")}
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <p className="text-sm whitespace-pre-wrap break-words">
                        {comment.content}
                      </p>
                      <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
                        <p className="text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(comment.created_at), {
                            addSuffix: true,
                            locale: dateLocale,
                          })}
                          {wasEdited && ` · ${t("jobs.commentEdited")}`}
                        </p>
                        <div className="flex items-center gap-1">
                          <button
                            type="button"
                            onClick={() => startEditing(comment)}
                            aria-label={t("jobs.editComment")}
                            className={`${COMMENT_ACTION_BUTTON} text-muted-foreground hover:bg-accent hover:text-foreground`}
                          >
                            <Pencil className="h-4 w-4" aria-hidden />
                          </button>
                          <button
                            type="button"
                            onClick={() => setPendingDelete(comment)}
                            aria-label={t("jobs.deleteComment")}
                            className={`${COMMENT_ACTION_BUTTON} text-muted-foreground hover:bg-destructive/10 hover:text-destructive`}
                          >
                            <Trash2 className="h-4 w-4" aria-hidden />
                          </button>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              );
            })}
          </div>
        )}

        <form onSubmit={onAddComment} className="space-y-2">
          <Textarea
            value={newComment}
            onChange={(e) => onChangeComment(e.target.value)}
            placeholder={t("jobs.commentPlaceholder")}
            className="flex-1"
            rows={3}
          />
          <Button
            type="submit"
            size="sm"
            disabled={!newComment.trim() || isAdding}
          >
            <MessageSquarePlus className="h-4 w-4 mr-2" />
            {t("jobs.addComment")}
          </Button>
        </form>
      </CardContent>

      {/* Deleting a comment cannot be undone, so the confirm quotes the comment
          being removed rather than asking about "this comment". */}
      <Dialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          // Escape and the backdrop must not pull the dialog out from under a
          // delete that is still in flight.
          if (!open && !isDeleting) setPendingDelete(null);
        }}
      >
        <DialogContent
          onClose={isDeleting ? undefined : () => setPendingDelete(null)}
        >
          <DialogHeader>
            <DialogTitle>{t("jobs.deleteComment")}</DialogTitle>
            <DialogDescription>
              {t("jobs.deleteCommentConfirm", {
                preview: pendingDelete ? commentPreview(pendingDelete.content) : "",
              })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-6">
            <Button
              variant="outline"
              onClick={() => setPendingDelete(null)}
              disabled={isDeleting}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={isDeleting}
            >
              {isDeleting ? t("common.deleting") : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
