// Only workflow identifiers are stored. Resume content remains on the server.
const fallback = new Map<string, string>();
const prefix = "jobber:resume-tailoring:";
export function workflowValue(key: string): string | null {
  try { return sessionStorage.getItem(prefix + key); } catch { return fallback.get(key) ?? null; }
}
export function saveWorkflowValue(key: string, value: string): void {
  fallback.set(key, value);
  try { sessionStorage.setItem(prefix + key, value); } catch { /* In-memory request guards still prevent double clicks. */ }
}
export function workflowRequestKey(key: string, fresh = false): string {
  const existing = fresh ? null : workflowValue(key);
  if (existing) return existing;
  const value = crypto.randomUUID();
  saveWorkflowValue(key, value);
  return value;
}
