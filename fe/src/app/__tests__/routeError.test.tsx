import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { RouteError } from "../RouteError";
import { routes as appRoutes } from "../router";
import { GlobalErrorBoundary } from "@/shared/ui/GlobalErrorBoundary";

const captureException = vi.hoisted(() => vi.fn());

vi.mock("@sentry/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@sentry/react")>()),
  captureException,
}));

function CrashingPage(): never {
  throw new Error("page crashed");
}

function renderCrashingPage() {
  const router = createMemoryRouter(
    [
      {
        path: "/",
        errorElement: <RouteError />,
        children: [{ index: true, element: <CrashingPage /> }],
      },
    ],
    { initialEntries: ["/"] },
  );
  render(
    <GlobalErrorBoundary>
      <RouterProvider router={router} />
    </GlobalErrorBoundary>,
  );
}

describe("a crash inside a page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("reaches Sentry instead of stopping at the router's own boundary", () => {
    renderCrashingPage();

    expect(captureException).toHaveBeenCalledOnce();
    expect(captureException.mock.calls[0]?.[0]).toMatchObject({
      message: "page crashed",
    });
  });

  it("shows the app's error page, not the router's raw message", () => {
    renderCrashingPage();

    expect(
      screen.queryByText("Unexpected Application Error!"),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("page crashed")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1 })).toBeInTheDocument();
  });

  // Every page in the app sits under this route.
  it("is wired on the route that holds the app", () => {
    const root = appRoutes.find((route) => route.path === "/");

    expect(root?.errorElement).toBeDefined();
  });
});
