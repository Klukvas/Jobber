import { useTranslation } from "react-i18next";
import { CopyButton } from "./CopyButton";

export function SetupSnippets({ token }: { token: string | null }) {
  const { t } = useTranslation();
  const url = `${window.location.origin}/api/mcp`;
  const secret = token ?? "<YOUR_TOKEN>";
  const snippets = [
    { name: "Claude Code", value: `claude mcp add --transport http courseopus ${url} --header "Authorization: Bearer ${secret}"` },
    { name: t("settings.aiAssistants.genericClient"), value: JSON.stringify({ mcpServers: { courseopus: { url, headers: { Authorization: `Bearer ${secret}` } } } }, null, 2) },
  ];
  return (
    <div className="space-y-4">
      <h3 className="font-semibold">{t("settings.aiAssistants.setup")}</h3>
      {snippets.map(({ name, value }) => (
        <div key={name} className="min-w-0 space-y-2">
          <h4 className="text-sm font-medium">{name}</h4>
          <pre className="max-w-full overflow-x-auto rounded-md border bg-muted/50 p-3 text-xs"><code>{value}</code></pre>
          <CopyButton key={token ? "created" : "placeholder"} value={value} label={t("settings.aiAssistants.copySnippet", { client: name })} />
        </div>
      ))}
    </div>
  );
}
