import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Sidebar } from "../Sidebar";
import { useSidebarStore } from "@/stores/sidebarStore";
import i18n from "@/shared/lib/i18n";

const permission = vi.hoisted(() => ({ allowed: false, isError: false }));
vi.mock("@/shared/hooks/useAdminAccess", () => ({
  useAdminAccess: () => ({
    data: { allowed: permission.allowed },
    isError: permission.isError,
  }),
}));
vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({ plan: "free" }),
}));
vi.mock("@/features/onboarding/useOnboarding", () => ({
  useOnboardingHighlight: () => null,
}));
beforeEach(async () => {
  await i18n.changeLanguage("en");
  permission.allowed = false;
  permission.isError = false;
  useSidebarStore.setState({ isExpanded: true, isMobileOpen: false });
});
afterEach(cleanup);
function mount() {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}
it("hides admin navigation for denied and failed access checks", () => {
  mount();
  expect(
    screen.queryByRole("link", { name: "Administration" }),
  ).not.toBeInTheDocument();
});
it("links the admin panel from desktop navigation", () => {
  permission.allowed = true;
  mount();
  expect(screen.getByRole("link", { name: "Administration" })).toHaveAttribute(
    "href",
    "/app/admin",
  );
});
it("includes admin navigation in the same mobile drawer", () => {
  permission.allowed = true;
  useSidebarStore.setState({ isMobileOpen: true });
  mount();
  expect(screen.getByRole("link", { name: "Administration" })).toHaveAttribute(
    "href",
    "/app/admin",
  );
});
it("removes a cached admin link after verification fails", () => {
  permission.allowed = true;
  permission.isError = true;
  mount();
  expect(
    screen.queryByRole("link", { name: "Administration" }),
  ).not.toBeInTheDocument();
});
