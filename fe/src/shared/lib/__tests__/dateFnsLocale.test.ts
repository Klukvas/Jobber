import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";

const mockLanguage = vi.hoisted(() => ({ value: "en" }));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    i18n: {
      get language() {
        return mockLanguage.value;
      },
    },
  }),
}));

import { useDateLocale } from "../dateFnsLocale";
import { enUS, uk, ru } from "date-fns/locale";

describe("useDateLocale", () => {
  it("returns enUS locale for 'en' language", () => {
    mockLanguage.value = "en";
    const { result } = renderHook(() => useDateLocale());
    expect(result.current).toBe(enUS);
  });

  it("returns uk locale for 'ua' language", () => {
    mockLanguage.value = "ua";
    const { result } = renderHook(() => useDateLocale());
    expect(result.current).toBe(uk);
  });

  it("returns ru locale for 'ru' language", () => {
    mockLanguage.value = "ru";
    const { result } = renderHook(() => useDateLocale());
    expect(result.current).toBe(ru);
  });

  it("falls back to enUS for unknown language", () => {
    mockLanguage.value = "fr";
    const { result } = renderHook(() => useDateLocale());
    expect(result.current).toBe(enUS);
  });

  it("falls back to enUS for empty string language", () => {
    mockLanguage.value = "";
    const { result } = renderHook(() => useDateLocale());
    expect(result.current).toBe(enUS);
  });
});
