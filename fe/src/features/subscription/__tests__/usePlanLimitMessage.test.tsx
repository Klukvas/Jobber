import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import i18n from "@/shared/lib/i18n";
import { usePlanLimitMessage } from "../usePlanLimitMessage";

const subscription = vi.hoisted(() => ({
  limits: {
    max_jobs: 25,
    max_resumes: 1,
    max_ai_requests: 3,
    max_resume_builders: 2,
    max_cover_letters: 0,
  },
  usage: {
    jobs: 25,
    resumes: 1,
    ai_requests: 3,
    resume_builders: 2,
    cover_letters: 0,
  },
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => subscription,
}));

function message(resource: Parameters<ReturnType<typeof usePlanLimitMessage>>[0]) {
  const { result } = renderHook(() => usePlanLimitMessage());
  return result.current(resource);
}

afterEach(async () => {
  await i18n.changeLanguage("en");
});

describe("usePlanLimitMessage", () => {
  it("states used of limit with the resource noun, in English", async () => {
    await i18n.changeLanguage("en");
    expect(message("jobs")).toBe(
      "You've reached the limit of your current plan (25 of 25 tracked jobs).",
    );
    expect(message("resumes")).toBe(
      "You've reached the limit of your current plan (1 of 1 uploaded resume).",
    );
  });

  it("says cover letters are not part of a plan that has none", async () => {
    await i18n.changeLanguage("en");
    expect(message("cover_letters")).toBe(
      "Cover letters are not included in your current plan.",
    );
  });

  it.each(["uk", "ru"] as const)("is localised in %s", async (language) => {
    await i18n.changeLanguage(language);
    const text = message("jobs");
    expect(text).toContain("25");
    expect(text).not.toContain("planLimit");
    expect(text).not.toContain("You've reached");
  });
});
