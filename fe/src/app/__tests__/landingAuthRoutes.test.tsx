import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { routes } from "../router";
import { useAuthStore } from "@/stores/authStore";

/**
 * `/login`, `/register` and `/forgot-password` are the landing page with a
 * modal over it. They used to sit one level deeper in the route tree than the
 * landing page itself, so every open and close moved the page component to a
 * different depth, React rebuilt it, and the control the visitor had just
 * pressed was replaced by a brand new node. Escape then handed focus to a
 * detached element — which silently drops it on `<body>`, leaving a keyboard
 * user at the top of the document with no idea where they are.
 *
 * These run against the real route table for that reason: the defect lived in
 * the shape of the tree, not in any one component.
 */
describe("auth modal routes", () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  function renderRouter(path: string) {
    const router = createMemoryRouter(routes, { initialEntries: [path] });
    // The auth modals talk to the API through react-query; nothing in these
    // tests submits anything, but the hooks still need a client.
    render(
      <QueryClientProvider client={new QueryClient()}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    );
    return router;
  }

  function renderLanding() {
    renderRouter("/");
    // The landing navbar. Everything else is queried inside it, so the mobile
    // copies of the same controls cannot be picked up by accident.
    return within(screen.getByRole("navigation"));
  }

  it("keeps the landing page mounted while a modal route is open", async () => {
    const user = userEvent.setup();
    const navbar = renderLanding();
    const landing = screen.getByRole("navigation");

    await user.click(navbar.getByRole("button", { name: "auth.login" }));

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    // The same node, not an equal one: a rebuilt page loses scroll position,
    // the forced-dark body class and — the reason this matters — the opener.
    expect(screen.getByRole("navigation")).toBe(landing);
  });

  it("returns focus to the opener when Escape closes the modal", async () => {
    const user = userEvent.setup();
    const navbar = renderLanding();
    const login = navbar.getByRole("button", { name: "auth.login" });

    await user.click(login);
    await user.keyboard("{Escape}");

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(document.activeElement).toBe(login);
  });

  it("returns focus to the opener after switching between auth modals", async () => {
    const user = userEvent.setup();
    const navbar = renderLanding();
    const login = navbar.getByRole("button", { name: "auth.login" });

    await user.click(login);
    // The "no account yet?" control inside the login modal, which navigates to
    // /register and takes its own opener down with it.
    const dialog = screen.getByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "auth.register" }));
    await user.keyboard("{Escape}");

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(document.activeElement).toBe(login);
  });

  it("returns focus to the opener from the forgot-password modal", async () => {
    const user = userEvent.setup();
    const navbar = renderLanding();
    const login = navbar.getByRole("button", { name: "auth.login" });

    await user.click(login);
    await user.click(
      within(screen.getByRole("dialog")).getByRole("button", {
        name: "auth.forgotPassword",
      }),
    );
    await user.keyboard("{Escape}");

    expect(document.activeElement).toBe(login);
  });

  it("sends a signed-in visitor away from an auth route", async () => {
    useAuthStore.setState({ isAuthenticated: true });

    const router = renderRouter("/login");

    // /app then redirects on to the app's own landing route.
    expect(router.state.location.pathname).toMatch(/^\/app/);
  });
});
