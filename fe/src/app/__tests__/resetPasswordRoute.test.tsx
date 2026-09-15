import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { routes } from "../router";
import { useAuthStore } from "@/stores/authStore";

const resetPassword = vi.hoisted(() => vi.fn());

vi.mock("@/services/authService", () => ({
  authService: {
    resetPassword,
    login: vi.fn(),
    register: vi.fn(),
    forgotPassword: vi.fn(),
    logout: vi.fn(),
  },
}));

/**
 * The tests query i18n keys rather than English copy: this suite mounts the
 * real route table without an initialised i18n instance, so `t()` returns the
 * key. That is exactly what the rest of the routing tests do.
 *
 * `/reset-password` used to be a `<Navigate to="/" replace />`.
 *
 * The reset email carries a six-digit code rather than a link, so the modal on
 * the landing page is the ordinary way through — but `POST /auth/reset-password`
 * takes `email` + `code` + the new password, and that is exactly what this URL
 * carries. Anyone who arrived on it, from a bookmark or a hand-built link, was
 * bounced to a page that could not use what they were holding.
 *
 * Run against the real route table: what was wrong was the table.
 */
describe("the /reset-password route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  function renderAt(path: string) {
    const router = createMemoryRouter(routes, { initialEntries: [path] });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    );
    return router;
  }

  it("renders the reset form for a link carrying an email and a code", async () => {
    const router = renderAt("/reset-password?email=a%40b.com&code=123456");

    expect(await screen.findByLabelText("auth.newPassword")).toBeInTheDocument();
    // Still on the URL the customer opened — not sent back to the landing page.
    expect(router.state.location.pathname).toBe("/reset-password");
  });

  it("names a link it cannot use instead of pretending to work", async () => {
    renderAt("/reset-password");

    expect(await screen.findByRole("heading", { level: 1 })).toHaveTextContent(
      "auth.invalidResetLink",
    );
  });

  it("keeps itself out of the index", async () => {
    renderAt("/reset-password?email=a%40b.com&code=123456");
    await screen.findByLabelText("auth.newPassword");

    await waitFor(() =>
      expect(
        document.querySelector('meta[name="robots"]')?.getAttribute("content"),
      ).toContain("noindex"),
    );
  });
});
