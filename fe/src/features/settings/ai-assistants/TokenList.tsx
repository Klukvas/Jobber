import { useEffect, useRef, useState } from "react";
import { useMutation, type UseQueryResult } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { apiTokensService, type ApiToken } from "@/services/apiTokensService";
import { ApiError } from "@/services/api";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { Button } from "@/shared/ui/Button";
import { ErrorState } from "@/shared/ui/ErrorState";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from "@/shared/ui/Dialog";
import { ListLoading } from "./ListState";
import { useNow } from "./useNow";

type Props = {
  query: UseQueryResult<ApiToken[], Error>;
  onRevoked: () => Promise<void>;
  onPaidFeature: () => void;
};
type TokenRowProps = { token: ApiToken; now: number; onRevoke: () => void };

function TokenRow({ token, now, onRevoke }: TokenRowProps) {
  const { t } = useTranslation();
  const locale = useDateLocale();
  const status = token.revoked_at ? "revoked"
    : token.expires_at && new Date(token.expires_at).getTime() <= now ? "expired"
    : "active";
  const dates = [
    ["created", token.created_at],
    ["lastUsed", token.last_used_at],
    ["expires", token.expires_at],
  ] as const;
  return (
    <li className={`rounded-lg border p-4 ${status === "revoked" ? "bg-muted/30 text-muted-foreground" : ""}`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 max-w-full space-y-1">
          <p className="break-words font-medium">{token.name}</p>
          <code className="text-xs text-muted-foreground">{token.token_prefix}…</code>
        </div>
        <span className="rounded border px-2 py-1 text-xs">
          {t(`settings.aiAssistants.${status}`)}
        </span>
      </div>
      <dl className="my-3 grid gap-2 text-sm tabular-nums sm:grid-cols-3">
        {dates.map(([label, value]) => (
          <div key={label}>
            <dt className="text-muted-foreground">{t(`settings.aiAssistants.${label}`)}</dt>
            <dd>{value ? format(new Date(value), "PP", { locale }) : t("settings.aiAssistants.never")}</dd>
          </div>
        ))}
      </dl>
      {status !== "revoked" && (
        <Button variant="outline" size="sm"
          aria-label={t("settings.aiAssistants.revokeNamed", { name: token.name })}
          onClick={onRevoke}>
          {t("settings.aiAssistants.revoke")}
        </Button>
      )}
    </li>
  );
}

export function TokenList({ query, onRevoked, onPaidFeature }: Props) {
  const { t } = useTranslation();
  const now = useNow();
  const [target, setTarget] = useState<ApiToken | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const focusFrame = useRef<number | null>(null);
  useEffect(() => () => {
    if (focusFrame.current !== null) cancelAnimationFrame(focusFrame.current);
  }, []);
  const tokens = query.data ?? [];
  const revoke = useMutation({
    mutationFn: apiTokensService.revoke,
    onSuccess: async () => {
      await onRevoked();
      setTarget(null);
      // Run after the dialog restores its trigger, which the refresh may remove.
      focusFrame.current = requestAnimationFrame(() => heading.current?.focus());
    },
    onError: (error) => {
      if (error instanceof ApiError && error.code === "PAID_FEATURE") onPaidFeature();
    },
  });
  const revokeErrorKey = revoke.error instanceof ApiError && revoke.error.code === "API_TOKEN_NOT_FOUND"
    ? "settings.aiAssistants.notFound" : "settings.aiAssistants.revokeError";

  return (
    <section className="space-y-3" aria-labelledby="mcp-tokens-title">
      <h3 ref={heading} tabIndex={-1} id="mcp-tokens-title" className="rounded font-semibold focus:outline-none focus:ring-2 focus:ring-ring">{t("settings.aiAssistants.tokens")}</h3>
      {query.isError && (
        <ErrorState message={t("settings.aiAssistants.listError")} onRetry={() => void query.refetch()} />
      )}
      {query.isPending ? <ListLoading kind="tokens" /> : tokens.length ? (
        <ul className="space-y-3">
          {tokens.map((token) => (
            <TokenRow key={token.id} token={token} now={now} onRevoke={() => {
              revoke.reset();
              setTarget(token);
            }} />
          ))}
        </ul>
      ) : !query.isError && (
        <p className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
          {t("settings.aiAssistants.tokensEmpty")}
        </p>
      )}
      <Dialog open={target !== null} onOpenChange={(open) => {
        if (!open && !revoke.isPending) setTarget(null);
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("settings.aiAssistants.revokeTitle")}</DialogTitle>
            <DialogDescription>
              {t("settings.aiAssistants.revokeDescription", { name: target?.name })}
            </DialogDescription>
          </DialogHeader>
          {revoke.isError && <p role="alert" className="mt-3 text-sm text-destructive">{t(revokeErrorKey)}</p>}
          <DialogFooter className="mt-4">
            <Button variant="outline" disabled={revoke.isPending} onClick={() => setTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" disabled={revoke.isPending} onClick={() => {
              if (target && !revoke.isPending) revoke.mutate(target.id);
            }}>
              {t(`settings.aiAssistants.${revoke.isPending ? "revoking" : "revoke"}`)}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
