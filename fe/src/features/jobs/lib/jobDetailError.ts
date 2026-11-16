import { ApiError } from "@/services/api";

/** Only known failure classes become user-facing copy; server text stays private. */
export function jobDetailErrorKey(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 404 || error.code === "JOB_NOT_FOUND") return "jobs.notFound";
    if (error.status === 400 || error.code === "INVALID_JOB_ID") return "jobs.invalidJobId";
    if (error.status === 401 || error.code === "UNAUTHORIZED") return "jobs.loadUnauthorized";
    if (error.status === 403 || error.code === "FORBIDDEN") return "jobs.loadForbidden";
  }
  return "jobs.loadError";
}
