import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, Link, RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { RootLayout } from "../layouts/RootLayout";
import { routes as appRoutes } from "../router";
import { LANDING_BODY_CLASS, isLandingPath } from "../landingBodyClass";
import { useAuthStore } from "@/stores/authStore";

vi.mock("@/shared/lib/gtag", () => ({
  initGA4: vi.fn(),
  setGA4Disabled: vi.fn(),
  trackGA4PageView: vi.fn(),
}));
vi.mock("@/shared/lib/posthog", () => ({
  initPostHog: vi.fn(),
  trackPageView: vi.fn(),
}));

/**
 * The landing class used to be owned by the landing page: added on mount,
 * removed on unmount. Production never runs it that way. `dist/index.html` is
 * the prerendered landing page *and* the SPA fallback, so a direct load or a
 * refresh of `/app/*` was served `<body class="landing-page">` with an app
 * route about to render into it — the landing page never mounted, its cleanup
 * never ran, and the app's light theme was drawn on the landing page's
 * forced-dark background.
 *
 * Every test here therefore starts from a document that *already* carries the
 * class, which is what the fallback shell hands the app. Client navigation is
 * covered too, but it was never the failing case.
 */
describe("the landing body class", () => {
  /** What the production fallback shell serves for every route. */
  function serveFallbackShell(): void {
    document.body.classList.add(LANDING_BODY_CLASS);
  }

  function hasLandingClass(): boolean {
    return document.body.classList.contains(LANDING_BODY_CLASS);
  }

  beforeEach(() => {
    document.body.className = "";
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  afterEach(() => {
    document.body.className = "";
  });

  describe("isLandingPath", () => {
    it.each(["/", "/login", "/register", "/forgot-password", "/login/"])(
      "treats %s as landing presentation",
      (path) => {
        expect(isLandingPath(path)).toBe(true);
      },
    );

    it.each(["/app", "/app/jobs", "/blog", "/print/resume", "/nope"])(
      "treats %s as an ordinary route",
      (path) => {
        expect(isLandingPath(path)).toBe(false);
      },
    );
  });

  describe("mounted on a shell that shipped the class", () => {
    /** A stand-in route tree: the real one guards `/app/*` behind auth. */
    const routes = [
      {
        path: "/",
        element: <RootLayout />,
        children: [
          {
            index: true,
            element: <Link to="/app/jobs">to app</Link>,
          },
          { path: "app/jobs", element: <p>jobs</p> },
          { path: "*", element: <p>not found</p> },
        ],
      },
    ];

    function renderAt(path: string) {
      return render(
        <RouterProvider
          router={createMemoryRouter(routes, { initialEntries: [path] })}
        />,
      );
    }

    it("drops it on a direct load of an app route", () => {
      serveFallbackShell();

      renderAt("/app/jobs");

      expect(screen.getByText("jobs")).toBeInTheDocument();
      expect(hasLandingClass()).toBe(false);
    });

    it("drops it on the 404 shell", () => {
      serveFallbackShell();

      renderAt("/nowhere");

      expect(screen.getByText("not found")).toBeInTheDocument();
      expect(hasLandingClass()).toBe(false);
    });

    it("keeps it on the landing route", () => {
      serveFallbackShell();

      renderAt("/");

      expect(hasLandingClass()).toBe(true);
    });

    it("removes it when navigating off the landing page", async () => {
      const user = userEvent.setup();
      renderAt("/");
      expect(hasLandingClass()).toBe(true);

      await user.click(screen.getByRole("link", { name: "to app" }));

      expect(screen.getByText("jobs")).toBeInTheDocument();
      expect(hasLandingClass()).toBe(false);
    });

    it("removes it when the app unmounts", () => {
      const { unmount } = renderAt("/");
      expect(hasLandingClass()).toBe(true);

      unmount();

      expect(hasLandingClass()).toBe(false);
    });
  });

  describe("on the real route table", () => {
    function renderAt(path: string) {
      render(
        <QueryClientProvider client={new QueryClient()}>
          <RouterProvider
            router={createMemoryRouter(appRoutes, { initialEntries: [path] })}
          />
        </QueryClientProvider>,
      );
    }

    it("styles the landing page", () => {
      renderAt("/");

      expect(hasLandingClass()).toBe(true);
    });

    // The auth routes are the landing page with a modal over it, so the shell
    // they are served in is styled the same way.
    it("styles the auth modal routes", () => {
      renderAt("/login");

      expect(screen.getByRole("dialog")).toBeInTheDocument();
      expect(hasLandingClass()).toBe(true);
    });
  });
});
