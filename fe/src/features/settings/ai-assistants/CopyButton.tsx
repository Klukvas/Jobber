import { useCallback, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";
import { Textarea } from "@/shared/ui/Textarea";

function selectForCopy(field: HTMLTextAreaElement | null): void {
  field?.focus();
  field?.select();
}

export function CopyButton({ value, label }: { value: string; label: string }) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<"idle" | "copied" | "error">("idle");
  const fallback = useRef<HTMLTextAreaElement | null>(null);
  const statusId = useId();
  const registerFallback = useCallback((field: HTMLTextAreaElement | null) => {
    fallback.current = field;
    selectForCopy(field);
  }, []);
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setStatus("copied");
    } catch {
      setStatus("error");
      selectForCopy(fallback.current);
    }
  }
  return (
    <div className="space-y-1">
      <Button type="button" variant="outline" size="sm" className="h-auto min-h-9 max-w-full whitespace-normal py-2 max-sm:min-h-11" aria-label={label} onClick={copy}>
        {status === "copied" ? t("settings.aiAssistants.copied") : label}
      </Button>
      <span id={statusId} role="status" className="block text-xs text-muted-foreground">
        {status === "copied" && t("settings.aiAssistants.copied")}
        {status === "error" && t("settings.aiAssistants.copyError")}
      </span>
      {status === "error" && (
        <Textarea
          ref={registerFallback}
          readOnly
          value={value}
          aria-label={label}
          aria-describedby={statusId}
          className="font-mono text-xs"
        />
      )}
    </div>
  );
}
