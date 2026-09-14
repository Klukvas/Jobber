import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { LandingLayout } from "../LandingLayout";

let pathname = "/";
let isAuthenticated = false;

vi.mock("react-router-dom", () => ({
  Outlet: () => <div data-testid="outlet">Outlet Content</div>,
  Navigate: ({ to }: { to: string }) => <div data-testid="navigate">{to}</div>,
  useLocation: () => ({ pathname, search: "", hash: "" }),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isAuthenticated }),
}));

function renderAt(path: string, authenticated: boolean) {
  pathname = path;
  isAuthenticated = authenticated;
  return render(<LandingLayout />);
}

describe("LandingLayout", () => {
  it("renders the landing page for a visitor", () => {
    renderAt("/", false);
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("renders the auth routes for a visitor", () => {
    renderAt("/login", false);
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  // The guard is per-path, not per-layout: the landing page now shares this
  // layout with the auth routes, and it stays open to signed-in visitors.
  it("keeps the landing page open to a signed-in user", () => {
    renderAt("/", true);
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("navigate")).not.toBeInTheDocument();
  });

  it.each(["/login", "/register", "/forgot-password"])(
    "sends a signed-in user away from %s",
    (path) => {
      renderAt(path, true);
      expect(screen.getByTestId("navigate")).toHaveTextContent("/app");
    },
  );
});
