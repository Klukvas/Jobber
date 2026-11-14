import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createInstance } from "i18next";
import { I18nextProvider } from "react-i18next";
import { useAuthStore } from "@/stores/authStore";
import en from "@/shared/locales/en.json";
import ru from "@/shared/locales/ru.json";
import uk from "@/shared/locales/uk.json";
import OAuthAuthorizePage from "./OAuthAuthorize";

const i18n = createInstance();
await i18n.init({ lng: "en", resources: { en: { translation: en }, ru: { translation: ru }, uk: { translation: uk } }, interpolation: { escapeValue: false } });
const search = "?response_type=code&client_id=client&redirect_uri=https%3A%2F%2Fassistant.example%2Fcb&code_challenge=challenge&code_challenge_method=S256&scope=mcp";
const preview = { client: { client_id: "client", name: "Assistant", redirect_host: "assistant.example", is_metadata_document: false }, scope: "mcp", resource: "https://cursopus.com/api/mcp", plan_allowed: true };
const originError = { error_code: "ORIGIN_NOT_ALLOWED", error_message: "Private backend details" };

beforeEach(async () => {
  await i18n.changeLanguage("en");
  useAuthStore.setState({ isAuthenticated: true, user: { id: "user-1", email: "user@example.com", name: "User", locale: "en", created_at: "2026-01-01T00:00:00Z" } });
  vi.stubGlobal("location", { ...window.location, assign: vi.fn() });
});
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe("OAuth trusted-origin error envelope", () => {
  it.each(["preview", "authorize"])("renders a useful message for a real 403 response during %s", async (stage) => {
    const fetch = vi.fn(async (request: Request) => {
      const rejected = stage === "preview" || request.method === "POST";
      return new Response(JSON.stringify(rejected ? originError : preview), {
        status: rejected ? 403 : 200, headers: { "content-type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetch);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/oauth/authorize${search}`]}><OAuthAuthorizePage /></MemoryRouter>
    </QueryClientProvider></I18nextProvider>);
    if (stage === "authorize") await userEvent.click(await screen.findByRole("button", { name: "Allow" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(en.oauth.originError);
    expect(screen.queryByText(originError.error_message)).not.toBeInTheDocument();
    expect(window.location.assign).not.toHaveBeenCalled();
    expect(fetch).toHaveBeenCalledTimes(stage === "preview" ? 1 : 2);
  });
  describe.each([
    ["en", en, "Too many attempts. Please try again shortly."],
    ["ru", ru, "Слишком много попыток. Попробуйте снова чуть позже."],
    ["uk", uk, "Забагато спроб. Спробуйте знову трохи пізніше."],
  ] as const)("rate limit in %s", (locale, messages, expected) => {
    it.each(["preview", "authorize"])("renders a friendly message for a real 429 during %s", async (stage) => {
      await i18n.changeLanguage(locale);
      vi.stubGlobal("fetch", vi.fn(async (request: Request) => {
        const rejected = stage === "preview" || request.method === "POST";
        return new Response(JSON.stringify(rejected ? { error_code: "RATE_LIMIT_EXCEEDED", error_message: "Private backend details" } : preview), {
          status: rejected ? 429 : 200, headers: { "content-type": "application/json" },
        });
      }));
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}>
        <MemoryRouter initialEntries={[`/oauth/authorize${search}`]}><OAuthAuthorizePage /></MemoryRouter>
      </QueryClientProvider></I18nextProvider>);
      if (stage === "authorize") await userEvent.click(await screen.findByRole("button", { name: messages.oauth.allow }));
      expect(await screen.findByRole("alert")).toHaveTextContent(expected);
      expect(screen.queryByText("Private backend details")).not.toBeInTheDocument();
      expect(window.location.assign).not.toHaveBeenCalled();
    });
  });
});
