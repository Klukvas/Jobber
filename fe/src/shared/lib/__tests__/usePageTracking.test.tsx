import { describe, expect, it, vi, beforeEach } from "vitest";
import { act, render } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { usePageTracking } from "../usePageTracking";

const trackers = vi.hoisted(() => ({
  trackPageView: vi.fn(),
  trackGA4PageView: vi.fn(),
}));

vi.mock("../posthog", () => ({ trackPageView: trackers.trackPageView }));
vi.mock("../gtag", () => ({ trackGA4PageView: trackers.trackGA4PageView }));

function Tracked() {
  usePageTracking();
  return null;
}

function renderAt(entry: string) {
  const router = createMemoryRouter([{ path: "*", element: <Tracked /> }], {
    initialEntries: [entry],
  });
  render(<RouterProvider router={router} />);
  return router;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("usePageTracking", () => {
  it("reports a cleaned address to both trackers", () => {
    renderAt("/reset-password?email=a%40b.co&code=123456");

    expect(trackers.trackPageView).toHaveBeenCalledExactlyOnceWith(
      "/reset-password",
    );
    expect(trackers.trackGA4PageView).toHaveBeenCalledExactlyOnceWith(
      "/reset-password",
    );
  });

  it("does not count typing in a search as new page views", async () => {
    const router = renderAt("/app/jobs");

    await act(() => router.navigate("/app/jobs?search=A"));
    await act(() => router.navigate("/app/jobs?search=Ac"));

    expect(trackers.trackPageView).toHaveBeenCalledOnce();
  });

  it("reports a real navigation", async () => {
    const router = renderAt("/app/jobs");

    await act(() => router.navigate("/app/resumes"));

    expect(trackers.trackPageView).toHaveBeenLastCalledWith("/app/resumes");
    expect(trackers.trackPageView).toHaveBeenCalledTimes(2);
  });
});
