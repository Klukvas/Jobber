import { useTranslation } from "react-i18next";
import { CopyButton } from "./CopyButton";

export function ConnectorUrl() {
  const { t } = useTranslation();
  const url = `${window.location.origin}/api/mcp`;
  return (
    <div className="space-y-3 rounded-lg border p-4">
      <p className="text-sm text-muted-foreground">{t("oauth.connectorDescription")}</p>
      <code className="block break-all text-sm">{url}</code>
      <CopyButton value={url} label={t("oauth.copyUrl")} />
    </div>
  );
}
