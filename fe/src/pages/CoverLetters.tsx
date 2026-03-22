import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation, getI18n } from "react-i18next";
import { Plus, Trash2, FileText, Pencil, Mail, Copy } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent } from "@/shared/ui/Card";
import { CoverLetterListSkeleton } from "@/shared/ui/PageSkeleton";
import { coverLetterService } from "@/services/coverLetterService";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { UpgradeBanner } from "@/features/subscription/components/UpgradeBanner";
import { ApiError } from "@/services/api";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/shared/ui/Dialog";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

const QUERY_KEY = ["cover-letters"];

export default function CoverLettersPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  usePageMeta({ titleKey: "coverLetter.title" });

  const { canCreate } = useSubscription();
  const limitReached = !canCreate("cover_letters");

  const [deleteTarget, setDeleteTarget] = useState<CoverLetterDTO | null>(null);

  const {
    data: coverLetters,
    isLoading,
    isError,
  } = useQuery({
    queryKey: QUERY_KEY,
    queryFn: () => coverLetterService.list(),
  });

  const createMutation = useMutation({
    mutationFn: () =>
      coverLetterService.create({
        title: t("coverLetter.untitled"),
        template: "professional",
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      navigate(`/app/cover-letters/${data.id}`);
    },
    onError: (error) => {
      if (error instanceof ApiError && error.code === "PLAN_LIMIT_REACHED") {
        queryClient.invalidateQueries({ queryKey: ["subscription"] });
        return;
      }
      showErrorNotification(
        error instanceof Error ? error.message : t("common.error"),
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => coverLetterService.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      setDeleteTarget(null);
      showSuccessNotification(t("coverLetter.deleted"));
    },
    onError: (error) => {
      showErrorNotification(
        error instanceof Error ? error.message : t("common.error"),
      );
    },
  });

  const duplicateMutation = useMutation({
    mutationFn: (id: string) => coverLetterService.duplicate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      showSuccessNotification(t("coverLetter.duplicated"));
    },
    onError: (error) => {
      if (error instanceof ApiError && error.code === "PLAN_LIMIT_REACHED") {
        queryClient.invalidateQueries({ queryKey: ["subscription"] });
        return;
      }
      showErrorNotification(
        error instanceof Error ? error.message : t("common.error"),
      );
    },
  });

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString(getI18n().language);
  };

  if (isLoading) {
    return <CoverLetterListSkeleton />;
  }

  if (isError) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">{t("coverLetter.title")}</h1>
        </div>
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <p className="text-sm text-destructive">{t("common.error")}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("coverLetter.title")}</h1>
        <Button
          onClick={() => createMutation.mutate()}
          disabled={createMutation.isPending || limitReached}
        >
          <Plus className="mr-2 h-4 w-4" />
          {t("coverLetter.create")}
        </Button>
      </div>

      {limitReached && <UpgradeBanner resource="cover_letters" />}

      {!coverLetters || coverLetters.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <Mail className="mb-4 h-12 w-12 text-muted-foreground" />
            <p className="mb-2 text-lg font-medium">{t("coverLetter.empty")}</p>
            <p className="mb-6 text-sm text-muted-foreground">
              {t("coverLetter.emptyDescription")}
            </p>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={limitReached}
            >
              <Plus className="mr-2 h-4 w-4" />
              {t("coverLetter.createFirst")}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {coverLetters.map((letter) => (
            <Card
              key={letter.id}
              className="group cursor-pointer transition-shadow hover:shadow-md"
              onClick={() => navigate(`/app/cover-letters/${letter.id}`)}
            >
              <CardContent className="p-4">
                <div className="mb-3 flex h-28 items-center justify-center rounded-md bg-muted">
                  <FileText className="h-12 w-12 text-muted-foreground" />
                </div>
                <div className="flex items-start justify-between">
                  <div className="min-w-0 flex-1">
                    <h3 className="truncate font-medium">{letter.title}</h3>
                    {letter.company_name && (
                      <p className="truncate text-xs text-primary">
                        {letter.company_name}
                      </p>
                    )}
                    <p className="text-sm text-muted-foreground">
                      {t("coverLetter.lastEdited", {
                        date: formatDate(letter.updated_at),
                      })}
                    </p>
                  </div>
                  <div className="ml-2 flex gap-1 opacity-100 sm:opacity-0 sm:transition-opacity sm:group-hover:opacity-100">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate(`/app/cover-letters/${letter.id}`);
                      }}
                      aria-label={t("common.edit")}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      disabled={limitReached}
                      onClick={(e) => {
                        e.stopPropagation();
                        duplicateMutation.mutate(letter.id);
                      }}
                      aria-label={t("common.duplicate")}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteTarget(letter);
                      }}
                      aria-label={t("common.delete")}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("coverLetter.deleteConfirmTitle")}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t("coverLetter.deleteConfirmDescription", {
              title: deleteTarget?.title,
            })}
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.id)
              }
              disabled={deleteMutation.isPending}
            >
              {t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
