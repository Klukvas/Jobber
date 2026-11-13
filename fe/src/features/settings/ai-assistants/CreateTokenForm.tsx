import { useEffect, useId, useRef, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { apiTokensService, type CreateApiToken } from "@/services/apiTokensService";
import { ApiError } from "@/services/api";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Select } from "@/shared/ui/Select";
import { CopyButton } from "./CopyButton";
import { SetupSnippets } from "./SetupSnippets";

type Props = { onCreated: () => void; onPaidFeature: () => void };

export function CreateTokenForm({ onCreated, onPaidFeature }: Props) {
  const { t } = useTranslation();
  const id = useId();
  const [name, setName] = useState("");
  const [expiry, setExpiry] = useState("90");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Plaintext stays in this mounted component only, never in a query/mutation cache.
  const [token, setToken] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);
  const panel = useRef<HTMLDivElement>(null);
  const nameInput = useRef<HTMLInputElement>(null);
  useEffect(() => () => request.current?.abort(), []);
  useEffect(() => { if (token) panel.current?.focus(); }, [token]);

  async function create(event: FormEvent) {
    event.preventDefault();
    if (request.current || token) return;
    const trimmed = name.trim();
    if (Array.from(trimmed).length < 1 || Array.from(trimmed).length > 100) {
      setError("nameError");
      nameInput.current?.focus();
      return;
    }
    const controller = new AbortController();
    request.current = controller;
    setPending(true);
    setError(null);
    try {
      const input: CreateApiToken = { name: trimmed, expires_in_days: expiry === "never" ? null : Number(expiry) as 30 | 90 | 365 };
      const result = await apiTokensService.create(input, { signal: controller.signal });
      if (controller.signal.aborted) return;
      setToken(result.token);
      setName("");
      onCreated();
    } catch (cause) {
      if (controller.signal.aborted) return;
      if (cause instanceof ApiError && cause.code === "PAID_FEATURE") onPaidFeature();
      else setError(cause instanceof ApiError && cause.code === "API_TOKEN_LIMIT_REACHED" ? "limitError" : cause instanceof ApiError && cause.code === "VALIDATION_ERROR" ? "validationError" : "createError");
    } finally {
      request.current = null;
      if (!controller.signal.aborted) setPending(false);
    }
  }

  function dismiss() {
    setToken(null);
    // The form reappears before focus is restored.
    requestAnimationFrame(() => nameInput.current?.focus());
  }

  return (
    <div className="min-w-0 space-y-6">
      {token ? (
        <div ref={panel} tabIndex={-1} role="region" aria-labelledby={`${id}-token-title`} className="space-y-3 rounded-lg border border-primary/40 bg-primary/5 p-4 focus:outline-none focus:ring-2 focus:ring-ring">
          <h3 id={`${id}-token-title`} className="font-semibold">{t("settings.aiAssistants.tokenCreated")}</h3>
          <p className="text-sm">{t("settings.aiAssistants.onceWarning")}</p>
          <code className="block break-all rounded border bg-background p-3 text-sm select-all">{token}</code>
          <div className="flex flex-wrap items-start gap-2">
            <CopyButton value={token} label={t("settings.aiAssistants.copyToken")} />
            <Button type="button" variant="outline" size="sm" className="h-auto min-h-9 whitespace-normal py-2 max-sm:min-h-11" onClick={dismiss}>{t("settings.aiAssistants.dismiss")}</Button>
          </div>
        </div>
      ) : (
        <form onSubmit={create} noValidate className="space-y-3" aria-label={t("settings.aiAssistants.create")}>
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor={`${id}-name`}>{t("settings.aiAssistants.name")}</Label>
              <Input ref={nameInput} id={`${id}-name`} value={name} onChange={(event) => setName(event.target.value)} required disabled={pending} aria-invalid={error === "nameError"} aria-describedby={error ? `${id}-error` : undefined} autoComplete="off" />
            </div>
            <div className="space-y-2">
              <Label htmlFor={`${id}-expiry`}>{t("settings.aiAssistants.expiry")}</Label>
              <Select className="max-sm:h-11" id={`${id}-expiry`} value={expiry} onChange={(event) => setExpiry(event.target.value)} disabled={pending}>
                {[30, 90, 365].map((days) => <option key={days} value={days}>{t("settings.aiAssistants.days", { count: days })}</option>)}
                <option value="never">{t("settings.aiAssistants.never")}</option>
              </Select>
            </div>
          </div>
          {error && <p id={`${id}-error`} role="alert" className="text-sm text-destructive">{t(`settings.aiAssistants.${error}`)}</p>}
          <Button type="submit" disabled={pending}>{t(`settings.aiAssistants.${pending ? "creating" : "create"}`)}</Button>
        </form>
      )}
      <SetupSnippets token={token} />
    </div>
  );
}
