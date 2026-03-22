import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { AuthLayout } from "../AuthLayout";

vi.mock("react-router-dom", () => ({
  Outlet: () => <div data-testid="outlet">Outlet Content</div>,
  Navigate: ({ to }: { to: string }) => (
    <div data-testid="navigate">{to}</div>
  ),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isAuthenticated: false }),
}));

describe("AuthLayout", () => {
  it("renders Outlet when not authenticated", () => {
    render(<AuthLayout />);
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });
});
