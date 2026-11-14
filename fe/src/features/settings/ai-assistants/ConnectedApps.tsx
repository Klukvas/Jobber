import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { oauthService, type OAuthGrant } from "@/services/oauthService";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { Button } from "@/shared/ui/Button";
import { ErrorState } from "@/shared/ui/ErrorState";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/shared/ui/Dialog";
import { ListLoading } from "./ListState";

function GrantRow({ grant, onRevoke }: { grant: OAuthGrant; onRevoke: () => void }) {
  const { t } = useTranslation();
  const locale = useDateLocale();
  const name = grant.client_name || t("oauth.unnamedApp");
  return (
    <li className="space-y-3 rounded-lg border p-4">
      <p className="break-words font-semibold">{name}</p>
      <p className="break-all font-mono text-sm" dir="ltr">{grant.redirect_host}</p>
      <dl className="grid gap-2 text-sm tabular-nums sm:grid-cols-2">
        {([["created", grant.created_at], ["lastUsed", grant.last_used_at]] as const).map(([label, value]) => (
          <div key={label}>
            <dt className="text-muted-foreground">{t(`settings.aiAssistants.${label}`)}</dt>
            <dd>{value ? <time dateTime={value}>{format(new Date(value), "PP", { locale })}</time> : t("settings.aiAssistants.never")}</dd>
          </div>
        ))}
      </dl>
      <Button variant="outline" size="sm" aria-label={t("oauth.revokeNamed", { name })} onClick={onRevoke}>{t("oauth.disconnect")}</Button>
    </li>
  );
}

function RevokeDialog({ target, pending, error, onClose, onRevoke }: {
  target: OAuthGrant | null; pending: boolean; error: boolean; onClose: () => void; onRevoke: () => void;
}) {
  const { t } = useTranslation();
  return (
    <Dialog open={target !== null} onOpenChange={(open) => { if (!open && !pending) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("oauth.revokeTitle")}</DialogTitle>
          <DialogDescription>{t("oauth.revokeDescription", { name: target?.client_name || t("oauth.unnamedApp") })}</DialogDescription>
        </DialogHeader>
        {error && <p role="alert" className="mt-3 text-sm text-destructive">{t("oauth.revokeError")}</p>}
        <DialogFooter className="mt-4">
          <Button variant="outline" disabled={pending} onClick={onClose}>{t("common.cancel")}</Button>
          <Button variant="destructive" disabled={pending} onClick={onRevoke}>{t(pending ? "settings.aiAssistants.revoking" : "oauth.disconnect")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function useConnectedApps(userId: string) {
  const client = useQueryClient();
  const key = ["oauth-grants", userId];
  const [target, setTarget] = useState<OAuthGrant | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const focusFrame = useRef<number | null>(null);
  const submitting = useRef(false);
  useEffect(() => () => { if (focusFrame.current !== null) cancelAnimationFrame(focusFrame.current); }, []);
  const query = useQuery({ queryKey: key, queryFn: ({ signal }) => oauthService.grants({ signal }), retry: false, refetchInterval: 60_000 });
  const revoke = useMutation({
    mutationFn: oauthService.revoke,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: key });
      setTarget(null);
      focusFrame.current = requestAnimationFrame(() => heading.current?.focus());
    },
    onSettled: () => { submitting.current = false; },
  });
  function disconnect() {
    if (!target || submitting.current) return;
    submitting.current = true;
    revoke.mutate(target.id);
  }
  function select(grant: OAuthGrant) {
    revoke.reset();
    setTarget(grant);
  }
  return { query, revoke, target, heading, disconnect, select, close: () => setTarget(null) };
}

export function ConnectedApps({ userId }: { userId: string }) {
  const { t } = useTranslation();
  const { query, revoke, target, heading, disconnect, select, close } = useConnectedApps(userId);
  const grants = query.data ?? [];
  return (
    <section aria-labelledby="mcp-apps-title" className="space-y-3">
      <h3 id="mcp-apps-title" ref={heading} tabIndex={-1} className="rounded font-semibold focus:outline-none focus:ring-2 focus:ring-ring">{t("oauth.connectedApps")}</h3>
      {query.isError && <ErrorState message={t("oauth.grantsError")} onRetry={() => void query.refetch()} />}
      {query.isPending ? <ListLoading kind="tokens" /> : grants.length ? (
        <ul className="space-y-3">{grants.map((grant) => <GrantRow key={grant.id} grant={grant} onRevoke={() => select(grant)} />)}</ul>
      ) : !query.isError && <p className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">{t("oauth.grantsEmpty")}</p>}
      <RevokeDialog target={target} pending={revoke.isPending} error={revoke.isError} onClose={close} onRevoke={disconnect} />
    </section>
  );
}
