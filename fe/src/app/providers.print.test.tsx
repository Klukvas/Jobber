import { afterEach, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import type { createMemoryRouter } from "react-router-dom";

const { router } = vi.hoisted(() => ({
  router: { current: null as ReturnType<typeof createMemoryRouter> | null },
}));
vi.mock("./router", async () => {
  const { createMemoryRouter } = await import("react-router-dom");
  router.current = createMemoryRouter([
    { path: "/", element: <div>Public page</div> },
    { path: "/print/resume", element: <div>Resume only</div> },
  ]);
  return { router: router.current };
});
vi.mock("./providers/AuthProvider", () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("sonner", () => ({
  Toaster: () => <div data-testid="toaster">Toast overlay</div>,
}));
vi.mock("@/shared/lib/consent", () => ({
  getStoredConsent: () => null,
  isAnalyticsConfigured: () => false,
  applyConsent: vi.fn(),
  CONSENT_RESET_EVENT: "jobber:cookie-consent-reset",
}));
import { Providers } from "./providers";
afterEach(() => {
  cleanup();
});
it("never mounts global overlays or consent layout padding on a fresh print page", async () => {
  await router.current?.navigate("/print/resume");
  render(<Providers />);
  expect(screen.getByText("Resume only")).toBeInTheDocument();
  expect(screen.queryByRole("region")).not.toBeInTheDocument();
  expect(screen.queryByTestId("toaster")).not.toBeInTheDocument();
  expect(document.body.style.paddingBottom).toBe("");
});
it("restores public consent behavior across SPA navigation to and from print", async () => {
  await router.current?.navigate("/");
  render(<Providers />);
  expect(screen.getByRole("region")).toBeInTheDocument();
  expect(document.body.style.paddingBottom).not.toBe("");
  await act(async () => {
    await router.current?.navigate("/print/resume");
  });
  expect(screen.queryByRole("region")).not.toBeInTheDocument();
  expect(screen.queryByTestId("toaster")).not.toBeInTheDocument();
  expect(document.body.style.paddingBottom).toBe("");
  await act(async () => {
    await router.current?.navigate("/");
  });
  expect(screen.getByRole("region")).toBeInTheDocument();
  expect(screen.getByTestId("toaster")).toBeInTheDocument();
});
