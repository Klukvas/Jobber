import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const LANGUAGE_KEY = "language";

type I18nModule = typeof import("@/shared/lib/i18n");

async function loadI18n(): Promise<I18nModule> {
  vi.resetModules();
  return import("@/shared/lib/i18n");
}

/**
 * The funnel and the share list used to render a number glued to a fixed noun,
 * so a single application read "1 applications" in English — and in Russian and
 * Ukrainian it read "1 заявки", which is wrong for 1, 2 and 5 in three
 * different ways. Counted through i18next, the whole set is one key.
 *
 * Driven through the real bundles and the real i18n setup: the plural forms
 * only resolve correctly because the Ukrainian bundle is registered under the
 * BCP 47 "uk", and a hand-rolled instance would not prove that.
 */
describe("counted application totals", () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => localStorage.clear());

  it("uses the singular for exactly one in English", async () => {
    const { default: i18n } = await loadI18n();

    expect(i18n.t("analytics.applicationsCount", { count: 1 })).toBe(
      "1 application",
    );
  });

  it.each([0, 2, 11, 25])(
    "uses the plural for %i in English",
    async (count) => {
      const { default: i18n } = await loadI18n();

      expect(i18n.t("analytics.applicationsCount", { count })).toBe(
        `${count} applications`,
      );
    },
  );

  // one / few / many, the three forms English does not have.
  it.each([
    [1, "1 заявка"],
    [2, "2 заявки"],
    [5, "5 заявок"],
    [21, "21 заявка"],
    [102, "102 заявки"],
    [111, "111 заявок"],
  ])("counts %i correctly in Russian", async (count, expected) => {
    localStorage.setItem(LANGUAGE_KEY, "ru");
    const { default: i18n } = await loadI18n();

    expect(i18n.t("analytics.applicationsCount", { count })).toBe(expected);
  });

  it.each([
    [1, "1 заявка"],
    [3, "3 заявки"],
    [5, "5 заявок"],
    [22, "22 заявки"],
  ])("counts %i correctly in Ukrainian", async (count, expected) => {
    localStorage.setItem(LANGUAGE_KEY, "uk");
    const { default: i18n } = await loadI18n();

    expect(i18n.t("analytics.applicationsCount", { count })).toBe(expected);
  });
});
