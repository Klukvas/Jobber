import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import type { ConsentPreview } from "@/services/oauthService";
import { Button } from "@/shared/ui/Button";
import { buttonVariants } from "@/shared/ui/buttonVariants";
import { UpgradeState } from "@/features/settings/ai-assistants/UpgradeState";

export function ConsentUpgrade({ onRefresh }: { onRefresh: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-5">
      <UpgradeState onClosed={onRefresh} />
      <Link to="/app" className={buttonVariants({ variant: "outline" })}>{t("oauth.goToApp")}</Link>
    </div>
  );
}

export function ConsentDetails({ preview, onRefresh }: { preview: ConsentPreview; onRefresh: () => void }) {
  const { t } = useTranslation();
  const loopback = /^(localhost|127\.0\.0\.1|\[::1\])(?::\d+)?$/i.test(preview.client.redirect_host);
  return (
    <div className="space-y-5">
      <h2 className="break-words text-xl font-semibold">{t("oauth.clientRequest", { name: preview.client.name || t("oauth.unnamedApp") })}</h2>
      <p className="text-sm text-muted-foreground">{t("oauth.unverified")}</p>
      <div className="rounded-lg border border-primary/40 bg-primary/5 p-4">
        <p className="text-sm text-muted-foreground">{t("oauth.redirectHost")}</p>
        <p className="mt-1 break-all font-mono text-lg font-semibold" dir="ltr">{preview.client.redirect_host}</p>
      </div>
      {loopback && <p role="note" className="rounded-lg border p-4 text-sm">{t("oauth.loopbackWarning")}</p>}
      <div>
        <h3 className="font-semibold">{t("oauth.permissionsTitle")}</h3>
        <ul className="mt-3 list-disc space-y-2 pl-5 text-sm text-muted-foreground">
          {["jobs", "resumes", "ai", "credits"].map((permission) => <li key={permission}>{t(`oauth.permissions.${permission}`)}</li>)}
        </ul>
      </div>
      <p className="text-sm text-muted-foreground">{t("oauth.revokeHint")}</p>
      {!preview.plan_allowed && <UpgradeState onClosed={onRefresh} />}
    </div>
  );
}

export function ConsentActions({ allowed, pending, error, onDecision }: {
  allowed: boolean; pending: boolean; error?: string; onDecision: (decision: "allow" | "deny") => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="mt-6 space-y-3" aria-busy={pending}>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      {pending && <p role="status" className="text-sm text-muted-foreground">{t("common.loading")}</p>}
      <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
        <Button variant="outline" disabled={pending} onClick={() => onDecision("deny")}>{t("oauth.deny")}</Button>
        <Button disabled={!allowed || pending} onClick={() => onDecision("allow")}>{t("oauth.allow")}</Button>
      </div>
    </div>
  );
}
