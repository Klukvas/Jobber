import { expect, it } from "vitest";
import { ApiError } from "@/services/api";
import { aiErrorKey } from "../useAIAvailability";
it.each([
  ["AI_JOB_DESCRIPTION_REQUIRED", "coverLetter.ai.jobDescriptionRequired"],
  ["AI_CANDIDATE_SOURCE_REQUIRED", "coverLetter.ai.resumeRequired"],
  ["AI_NO_CHANGES", "common.aiErrors.noChanges"],
  ["AI_RESULT_SAVE_FAILED", "common.aiErrors.resultSaveFailed"],
  ["INVALID_DATE_RANGE", "resumeBuilder.errors.invalidDateRange"],
  ["AI_INVALID_RESPONSE", "ai.invalidResponse"],
])("localizes %s without quoting server text", (code, key) => {
  expect(aiErrorKey(new ApiError("arbitrary server text", code, 422))).toBe(key);
});
