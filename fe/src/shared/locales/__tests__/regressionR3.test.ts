import { createInstance } from "i18next";
import { expect, it } from "vitest";
import en from "../en.json";
import ru from "../ru.json";
import uk from "../uk.json";

it.each([
  ["en", ["1 user", "2 users", "5 users", "21 users"]],
  ["ru", ["1 пользователь", "2 пользователя", "5 пользователей", "21 пользователь"]],
  ["uk", ["1 користувач", "2 користувачі", "5 користувачів", "21 користувач"]],
] as const)("uses admin count forms in %s", async (lng, expected) => {
  const i18n = createInstance();
  await i18n.init({ lng, resources: { en: { translation: en }, ru: { translation: ru }, uk: { translation: uk } } });
  expect([1, 2, 5, 21].map((count) => i18n.t("admin.total", { count }))).toEqual(expected);
});
it.each([ru, uk])("localizes job and builder resume terminology in both AI namespaces", (locale) => {
  expect(JSON.stringify([locale.jobs.tailoring, locale.jobs.interview])).not.toMatch(/\bJob\b|Builder Resume/);
});
