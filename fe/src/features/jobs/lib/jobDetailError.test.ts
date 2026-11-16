import { expect, it } from "vitest";
import { ApiError } from "@/services/api";
import { jobDetailErrorKey } from "./jobDetailError";

it.each([
  [400, "VALIDATION_ERROR", "jobs.invalidJobId"],
  [404, "JOB_NOT_FOUND", "jobs.notFound"],
  [401, "UNAUTHORIZED", "jobs.loadUnauthorized"],
  [403, "FORBIDDEN", "jobs.loadForbidden"],
  [500, "INTERNAL_ERROR", "jobs.loadError"],
])("maps HTTP %s to local copy without exposing backend text", (status, code, key) => {
  expect(jobDetailErrorKey(new ApiError("private backend text", code, status))).toBe(key);
});
it("hides unknown errors", () => {
  expect(jobDetailErrorKey(new Error("private backend text"))).toBe("jobs.loadError");
});
