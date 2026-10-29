export type CreditOperation = "tailor" | "cover_letter" | "match" | "resume_import" | "autofill_import" | "job_parse" | "bullet" | "summary" | "improve" | "ats" | "interview_plan" | "interview_feedback" | "interview_summary";

/** Only provider-start routes belong here; polling, applying and editing are free. */
export function creditOperation(path: string): CreditOperation | null {
  const route = path.split("?")[0].replace(/^\//, "");
  if (route === "match-score") return "match";
  if (route === "cover-letters/ai/generate") return "cover_letter";
  if (/^resume-builder\/import\/(text|pdf)$/.test(route)) return "resume_import";
  if (/^jobs\/[^/]+\/resume-tailoring\/import$/.test(route)) return "resume_import";
  if (/^jobs\/[^/]+\/resume-tailoring$/.test(route)) return "tailor";
  if (/^resume-builder\/[^/]+\/ats-check$/.test(route)) return "ats";
  if (route === "resume-builder/ai/suggest-bullets") return "bullet";
  if (route === "resume-builder/ai/suggest-summary") return "summary";
  if (route === "resume-builder/ai/improve-text") return "improve";
  if (/^jobs\/[^/]+\/interview-prep\/generate$/.test(route)) return "interview_plan";
  if (/^jobs\/[^/]+\/interview-prep\/sessions\/[^/]+\/answers\/[^/]+\/feedback$/.test(route)) return "interview_feedback";
  if (/^jobs\/[^/]+\/interview-prep\/sessions\/[^/]+\/summary$/.test(route)) return "interview_summary";
  if (/^resumes\/[^/]+\/autofill-profile$/.test(route)) return "autofill_import";
  if (route === "jobs/parse") return "job_parse";
  return null;
}

type PendingConsent = { readonly operation: CreditOperation; readonly finish: (accepted: boolean) => void };
let pending: PendingConsent | null = null;
let listeners: readonly (() => void)[] = [];
export const aiConsentStore = {
  subscribe(listener: () => void) {
    listeners = [...listeners, listener];
    return () => { listeners = listeners.filter((entry) => entry !== listener); };
  },
  getSnapshot: () => pending,
};
function changed() { listeners.forEach((listener) => listener()); }

function browserOrigin(): string {
  return `${window.location.href}:${window.history.state?.key ?? ""}`;
}

/** No API request is sent until the mounted confirmation explicitly accepts it. */
export async function confirmAIAction(path: string, signal?: AbortSignal): Promise<string | undefined> {
  const operation = creditOperation(path);
  if (!operation) return undefined;
  const origin = browserOrigin();
  if (signal?.aborted || pending) throw new DOMException("AI action cancelled", "AbortError");
  await new Promise<void>((resolve, reject) => {
    let settled = false;
    const abort = () => finish(false);
    const finish = (accepted: boolean) => {
      if (settled) return;
      settled = true;
      signal?.removeEventListener("abort", abort);
      pending = null;
      changed();
      if (accepted && browserOrigin() === origin) resolve();
      else reject(new DOMException("AI action cancelled", "AbortError"));
    };
    pending = { operation, finish };
    signal?.addEventListener("abort", abort, { once: true });
    changed();
  });
  if (signal?.aborted || browserOrigin() !== origin) throw new DOMException("AI action cancelled", "AbortError");
  return crypto.randomUUID();
}

export function isAIActionCancelled(error: unknown): boolean {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
