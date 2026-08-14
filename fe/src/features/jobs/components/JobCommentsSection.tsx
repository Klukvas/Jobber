import { useTranslation } from "react-i18next";
import { formatDistanceToNow } from "date-fns";
import { MessageSquarePlus } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { Textarea } from "@/shared/ui/Textarea";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import type { CommentDTO } from "@/shared/types/api";

interface JobCommentsSectionProps {
  readonly comments: readonly CommentDTO[];
  readonly newComment: string;
  readonly onChangeComment: (value: string) => void;
  readonly onAddComment: (e: React.FormEvent) => void;
  readonly isAdding: boolean;
}

export function JobCommentsSection({
  comments,
  newComment,
  onChangeComment,
  onAddComment,
  isAdding,
}: JobCommentsSectionProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">{t("jobs.comments")}</CardTitle>
      </CardHeader>
      <CardContent>
        {comments.length > 0 && (
          <div className="space-y-3 mb-4">
            {comments.map((comment) => (
              <div
                key={comment.id}
                className="rounded-lg border bg-muted/50 p-3"
              >
                <p className="text-sm whitespace-pre-wrap">{comment.content}</p>
                <p className="text-xs text-muted-foreground mt-2">
                  {formatDistanceToNow(new Date(comment.created_at), {
                    addSuffix: true,
                    locale: dateLocale,
                  })}
                </p>
              </div>
            ))}
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
    </Card>
  );
}
