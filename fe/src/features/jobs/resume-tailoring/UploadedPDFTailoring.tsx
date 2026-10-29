import { useAIConsentSignal } from "@/shared/hooks/useAIConsentSignal";
import { isAIActionCancelled } from "@/shared/lib/aiCredits";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { ApiError } from "@/services/api";
import { resumeTailoringService } from "@/services/resumeTailoringService";
import { Button } from "@/shared/ui/Button";
import { Label } from "@/shared/ui/Label";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import { tailoringErrorKey } from "./useResumeTailoring";
import { workflowRequestKey } from "./workflowSession";

export function UploadedPDFTailoring({ jobId, resumeId, disabled }: { jobId: string; resumeId: string; disabled: boolean }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const pending = useRef(false);
  const mounted = useRef(true);
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  const [terminalFailure, setTerminalFailure] = useState(false);
  const [busy, setBusy] = useState(false);
  const [errorKey, setErrorKey] = useState<string | null>(null);
  const [manual, setManual] = useState(false);
  const [text, setText] = useState("");
  const getConsentSignal = useAIConsentSignal();
  const importPDF = async () => {
    if (pending.current || disabled || (manual && !text.trim())) return;
    pending.current = true; setBusy(true); setErrorKey(null);
    const identity = `import:${jobId}:${resumeId}:${manual ? "manual:" + text.trim() : "pdf"}`;
    // The storage key must not contain the pasted content.
    try {
    const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(identity));
    const storageId = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
      const result = await resumeTailoringService.importUploaded(jobId, resumeId, workflowRequestKey(storageId, terminalFailure), manual ? text.trim() : undefined, { consentSignal: getConsentSignal() });
      void queryClient.invalidateQueries({ queryKey: resumeQueryKeys.builderList() });
      void queryClient.invalidateQueries({ queryKey: ["subscription"] });
      if (!mounted.current) return;
      navigate(`/app/resume-builder/${result.resume_builder.id}?tailoringJob=${encodeURIComponent(jobId)}&recognition=${encodeURIComponent(result.import_id)}`);
    } catch (error) {
      if (isAIActionCancelled(error)) return;
      if (!mounted.current) return;
      setTerminalFailure(error instanceof ApiError && ["AI_TIMEOUT", "AI_INVALID_RESPONSE", "AI_REQUEST_FAILED", "OPERATION_INTERRUPTED", "OPERATION_RESULT_DELETED"].includes(error.code));
      setErrorKey(error instanceof ApiError && error.code === "PDF_UNREADABLE" ? "pdfUnreadable" : tailoringErrorKey(error));
      if (error instanceof ApiError && error.code === "PDF_UNREADABLE") setManual(true);
    } finally { pending.current = false; if (mounted.current) setBusy(false); }
  };
  return <div className="space-y-3 rounded-lg border p-4">
    <p className="text-sm">{t("jobs.tailoring.pdf.disclosure")}</p>
    <p className="text-sm text-muted-foreground">{t("jobs.tailoring.pdf.usage")}</p>
    <p className="text-xs text-muted-foreground">{t("jobs.tailoring.pdf.cancelHint")}</p>
    <Button variant="ghost" disabled={busy} onClick={() => setManual((current) => !current)}>{t("jobs.tailoring.pdf.manual")}</Button>
    {manual && <div className="space-y-2"><Label htmlFor="tailoring-manual-text">{t("resumeBuilder.import.resumeText")}</Label><textarea id="tailoring-manual-text" className="min-h-48 w-full rounded-md border bg-background p-3 text-sm" value={text} maxLength={100000} disabled={busy} onChange={(event) => setText(event.target.value)} /></div>}
    {errorKey && <p role="alert" className="text-sm text-destructive">{t(`jobs.tailoring.errors.${errorKey}`)}</p>}
    <Button disabled={disabled || busy || (manual && !text.trim())} onClick={() => void importPDF()}>{busy && <Loader2 className="h-4 w-4 animate-spin" />}{t(terminalFailure ? "jobs.tailoring.newAttempt" : "jobs.tailoring.pdf.importReview")}</Button>
  </div>;
}
