import { act, render } from "@testing-library/react";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useScrollToTopOnNavigate } from "../useScrollToTopOnNavigate";

// "/login" is drawn over "/", as the auth modals are over the landing page.
const screenOf = (pathname: string) => (pathname === "/login" ? "/" : pathname);

function Root() {
  useScrollToTopOnNavigate(screenOf);
  return <Outlet />;
}

function renderRouter(initialPath: string) {
  const router = createMemoryRouter(
    [
      {
        path: "/",
        element: <Root />,
        children: [
          { index: true, element: <p>Landing</p> },
          { path: "login", element: <p>Landing</p> },
          { path: "features/applications", element: <p>Feature</p> },
        ],
      },
    ],
    { initialEntries: [initialPath] },
  );
  render(<RouterProvider router={router} />);
  return router;
}

describe("useScrollToTopOnNavigate", () => {
  let scrollTo: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    scrollTo = vi.fn();
    vi.stubGlobal("scrollTo", scrollTo);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("opens a page reached from the landing footer at its top", async () => {
    const router = renderRouter("/");

    await act(() => router.navigate("/features/applications"));

    expect(scrollTo).toHaveBeenCalledWith(0, 0);
  });

  it("leaves the page behind an auth modal where it was", async () => {
    const router = renderRouter("/");

    await act(() => router.navigate("/login"));
    await act(() => router.navigate("/"));

    expect(scrollTo).not.toHaveBeenCalled();
  });

  it("leaves Back to the browser", async () => {
    const router = renderRouter("/");
    await act(() => router.navigate("/features/applications"));
    scrollTo.mockClear();

    await act(() => router.navigate(-1));

    expect(scrollTo).not.toHaveBeenCalled();
  });

  it("lets a #section link scroll itself", async () => {
    const router = renderRouter("/features/applications");

    await act(() => router.navigate("/#faq"));

    expect(scrollTo).not.toHaveBeenCalled();
  });
});
