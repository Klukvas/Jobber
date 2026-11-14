import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useSingleFlightMutation } from "../useSingleFlightMutation";

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useSingleFlightMutation", () => {
  // Regression: a double-click created two jobs, companies, letters and
  // comments, because `isPending` reaches the button only after a re-render.
  it("ignores a second mutate while the first is in flight", async () => {
    let resolve: (value: string) => void = () => {};
    const mutationFn = vi.fn(
      () =>
        new Promise<string>((r) => {
          resolve = r;
        }),
    );
    const { result } = renderHook(
      () => useSingleFlightMutation({ mutationFn }),
      { wrapper },
    );

    act(() => {
      result.current.mutate();
      result.current.mutate();
    });

    await waitFor(() => expect(mutationFn).toHaveBeenCalled());
    await act(async () => resolve("done"));
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mutationFn).toHaveBeenCalledTimes(1);
  });

  it("accepts a new mutate once the previous one settled, even after an error", async () => {
    const mutationFn = vi
      .fn()
      .mockRejectedValueOnce(new Error("boom"))
      .mockResolvedValue("ok");
    const onSettled = vi.fn();
    const { result } = renderHook(
      () => useSingleFlightMutation({ mutationFn }),
      { wrapper },
    );

    act(() => result.current.mutate(undefined, { onSettled }));
    await waitFor(() => expect(result.current.isError).toBe(true));
    act(() => result.current.mutate(undefined));
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mutationFn).toHaveBeenCalledTimes(2);
    expect(onSettled).toHaveBeenCalledTimes(1);
  });
});
