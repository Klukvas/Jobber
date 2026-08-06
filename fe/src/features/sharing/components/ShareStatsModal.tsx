import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  OverviewAnalytics,
  FunnelAnalytics,
} from "@/services/analyticsService";
import {
  sharingService,
  buildShareUrl,
  type ShareDTO,
} from "@/services/sharingService";
import { ApiError } from "@/services/api";
import {
  showErrorNotification,
  showSuccessNotification,
} from "@/shared/lib/notifications";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { Skeleton } from "@/shared/ui/Skeleton";
import { Copy, Link2, Loader2, ShieldCheck, Trash2 } from "lucide-react";

interface ShareStatsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  overview?: OverviewAnalytics;
  funnel?: FunnelAnalytics;
}

async function copyShareLink(
  token: string,
  onSuccess: string,
  onError?: string,
) {
  try {
    await navigator.clipboard.writeText(buildShareUrl(token));
    showSuccessNotification(onSuccess);
  } catch {
    // Clipboard may be unavailable outside a direct user gesture (iOS
    // Safari). Without onError stay silent — the manual Copy button is
    // right next to the link.
    if (onError) {
      showErrorNotification(onError);
    }
  }
}

export function ShareStatsModal({
  open,
  onOpenChange,
  overview,
  funnel,
}: ShareStatsModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [createdShare, setCreatedShare] = useState<ShareDTO | null>(null);

  const sharesQuery = useQuery({
    queryKey: ["shares"],
    queryFn: () => sharingService.list(),
    enabled: open,
  });

  const createMutation = useMutation({
    mutationFn: () => sharingService.create(),
    onSuccess: (share) => {
      setCreatedShare(share);
      queryClient.invalidateQueries({ queryKey: ["shares"] });
      copyShareLink(share.token, t("sharing.linkCopied"));
    },
    onError: (error: Error) => {
      if (error instanceof ApiError && error.code === "SHARE_LIMIT_REACHED") {
        showErrorNotification(t("sharing.limitReached"));
        return;
      }
      showErrorNotification(error.message || t("sharing.createError"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => sharingService.remove(id),
    onSuccess: (_, id) => {
      if (createdShare?.id === id) {
        setCreatedShare(null);
      }
      queryClient.invalidateQueries({ queryKey: ["shares"] });
      showSuccessNotification(t("sharing.revoked"));
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("sharing.revokeError"));
    },
  });

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setCreatedShare(null);
    }
    onOpenChange(nextOpen);
  };

  const shares = sharesQuery.data ?? [];

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent onClose={() => handleOpenChange(false)}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Link2 className="h-5 w-5" />
            {t("sharing.modalTitle")}
          </DialogTitle>
          <DialogDescription>{t("sharing.modalDescription")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Consent preview: exactly what goes public */}
          <div className="rounded-md border p-3 space-y-2">
            <p className="text-sm font-medium">
              {t("sharing.whatWillBeShared")}
            </p>
            <ul className="text-sm text-muted-foreground space-y-0.5">
              <li>
                {t("analytics.overview.totalApplications")}:{" "}
                {overview?.total_applications ?? "—"}
              </li>
              <li>
                {t("analytics.overview.responseRate")}:{" "}
                {overview ? `${overview.response_rate}%` : "—"}
              </li>
              <li>
                {t("analytics.funnel.title")}:{" "}
                {funnel?.stages && funnel.stages.length > 0
                  ? funnel.stages.map((s) => s.stage_name).join(" → ")
                  : "—"}
              </li>
            </ul>
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <ShieldCheck className="h-3.5 w-3.5 shrink-0" />
              {t("sharing.privacyNote")}
            </p>
          </div>

          {createdShare && (
            <div className="rounded-md border border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-950 p-3 space-y-2">
              <p className="text-sm font-medium text-green-900 dark:text-green-100">
                {t("sharing.linkReady")}
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 truncate rounded bg-background px-2 py-1 text-xs">
                  {buildShareUrl(createdShare.token)}
                </code>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() =>
                    copyShareLink(
                      createdShare.token,
                      t("sharing.linkCopied"),
                      t("sharing.copyFailed"),
                    )
                  }
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}

          <Button
            type="button"
            className="w-full"
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending || !overview}
          >
            {createMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t("common.loading")}
              </>
            ) : (
              t("sharing.createButton")
            )}
          </Button>
          <p className="text-xs text-muted-foreground">
            {t("sharing.snapshotNote")}
          </p>

          {/* Existing links with revoke */}
          <div className="space-y-2">
            <p className="text-sm font-medium">{t("sharing.existingShares")}</p>
            {sharesQuery.isLoading ? (
              <Skeleton className="h-10 w-full" />
            ) : shares.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                {t("sharing.noShares")}
              </p>
            ) : (
              <ul className="space-y-2 max-h-48 overflow-y-auto">
                {shares.map((share) => (
                  <li
                    key={share.id}
                    className="flex items-center gap-2 rounded-md border p-2"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-mono">
                        {buildShareUrl(share.token)}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(share.created_at).toLocaleDateString()} ·{" "}
                        {share.snapshot.overview.total_applications}{" "}
                        {t("analytics.applications")}
                      </p>
                    </div>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() =>
                        copyShareLink(
                          share.token,
                          t("sharing.linkCopied"),
                          t("sharing.copyFailed"),
                        )
                      }
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => deleteMutation.mutate(share.id)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
