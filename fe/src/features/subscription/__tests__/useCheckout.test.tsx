import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

import { useCheckout } from "../useCheckout";
import { PRE_CHECKOUT_PLAN_KEY, readPreCheckoutPlan } from "../checkoutSignals";
import type { PopupCheckoutHandlers } from "../fastspringSbl";

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { PAYMENTS: true },
}));

const mockConfig = vi.hoisted(() => ({ value: undefined as unknown }));
const mockGetQueryData = vi.hoisted(() => vi.fn());
const mockCreateSession = vi.hoisted(() => vi.fn());
const mockOpenPopup = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: mockConfig.value }),
  useQueryClient: () => ({ getQueryData: mockGetQueryData }),
  // Minimal stand-in: calls the mutation function directly.
  useMutation: ({ mutationFn }: { mutationFn: (plan: string) => unknown }) => ({
    mutateAsync: mutationFn,
    isPending: false,
  }),
}));

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: {
    getCheckoutConfig: vi.fn(),
    createCheckoutSession: (plan: string) => mockCreateSession(plan),
  },
}));

vi.mock("@/features/subscription/fastspringSbl", () => ({
  openPopupCheckout: (options: unknown) => mockOpenPopup(options),
}));

const readyConfig = {
  provider: "fastspring",
  environment: "test",
  storefront: "fluxlab.test.onfastspring.com/popup-jobber",
  plans: ["pro", "enterprise"],
};

/** The handlers the hook handed to the popup on the last open. */
function popupHandlers(): PopupCheckoutHandlers {
  const options = mockOpenPopup.mock.calls.at(-1)?.[0] as {
    handlers: PopupCheckoutHandlers;
  };
  return options.handlers;
}

describe("useCheckout", () => {
  let assignSpy: ReturnType<typeof vi.fn>;
  let originalLocation: Location;

  beforeEach(() => {
    vi.clearAllMocks();
    mockConfig.value = readyConfig;
    mockCreateSession.mockResolvedValue({ session_id: "sess-1" });
    mockOpenPopup.mockResolvedValue(undefined);
    mockGetQueryData.mockReturnValue({ plan: "free" });
    sessionStorage.clear();

    // The whole point of the popup is that nothing navigates. Any assignment
    // here would be the full-page checkout coming back.
    assignSpy = vi.fn();
    originalLocation = window.location;
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, assign: assignSpy },
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: originalLocation,
    });
    sessionStorage.clear();
  });

  it("reports ready when the backend advertises plans and a storefront", () => {
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(true);
  });

  it("reports not ready when there is no config", () => {
    mockConfig.value = undefined;
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(false);
  });

  it("reports not ready when the backend advertises no plans", () => {
    mockConfig.value = { ...readyConfig, plans: [] };
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(false);
  });

  it("reports not ready when the backend advertises no storefront", () => {
    // An unopenable checkout must not be offered at all.
    mockConfig.value = { ...readyConfig, storefront: "" };
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(false);
  });

  it("opens the popup on this page instead of navigating anywhere", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockCreateSession).toHaveBeenCalledWith("pro");
    expect(mockOpenPopup).toHaveBeenCalledTimes(1);
    expect(mockOpenPopup.mock.calls[0][0]).toMatchObject({
      storefront: readyConfig.storefront,
      environment: "test",
      sessionId: "sess-1",
    });
    expect(assignSpy).not.toHaveBeenCalled();
  });

  it("passes the backend's session id and no checkout URL", async () => {
    // The session id is the entire cart; a URL is something to navigate to.
    mockCreateSession.mockResolvedValue({
      session_id: "sess-9",
      expires_at: "2026-01-01T00:00:00Z",
    });
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    const options = mockOpenPopup.mock.calls[0][0] as Record<string, unknown>;
    expect(options.sessionId).toBe("sess-9");
    // `returnFocusTo` is the app's own control, handed to the app's own
    // keyboard handling — nothing about it reaches the provider.
    expect(Object.keys(options).sort()).toEqual([
      "environment",
      "handlers",
      "returnFocusTo",
      "sessionId",
      "storefront",
    ]);
  });

  it("sends no user identity to the provider from the browser", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    // The plan is the only thing the browser gets to choose; the buyer comes
    // from the authenticated session on the backend.
    expect(mockCreateSession).toHaveBeenCalledTimes(1);
    expect(mockCreateSession.mock.calls[0]).toEqual(["pro"]);
  });

  it("persists the baseline plan before the popup opens", async () => {
    mockGetQueryData.mockReturnValue({ plan: "pro" });
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("enterprise"));

    expect(readPreCheckoutPlan()).toBe("pro");
  });

  it("defaults the baseline to free when no subscription is cached", async () => {
    mockGetQueryData.mockReturnValue(undefined);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(readPreCheckoutPlan()).toBe("free");
  });

  it("stays pending while the popup is open", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    expect(result.current.isPending).toBe(true);

    await act(async () => popupHandlers().onClose({ completed: true }));
    expect(result.current.isPending).toBe(false);
  });

  it("ignores a second click while a checkout is already starting", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(async () => {
      await Promise.all([
        result.current.openCheckout("pro"),
        result.current.openCheckout("pro"),
      ]);
    });

    expect(mockCreateSession).toHaveBeenCalledTimes(1);
    expect(mockOpenPopup).toHaveBeenCalledTimes(1);
  });

  it("announces a completed purchase without claiming it as one", async () => {
    const completed = vi.fn();
    window.addEventListener("jobber:checkout-completed", completed);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await act(async () => popupHandlers().onClose({ completed: true }));

    // The event only starts polling — the baseline stays for the layout to read.
    expect(completed).toHaveBeenCalledTimes(1);
    expect(readPreCheckoutPlan()).toBe("free");
    expect(result.current.error).toBeNull();
    window.removeEventListener("jobber:checkout-completed", completed);
  });

  it("clears the baseline and celebrates nothing when the buyer cancels", async () => {
    const completed = vi.fn();
    window.addEventListener("jobber:checkout-completed", completed);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await act(async () => popupHandlers().onClose({ completed: false }));

    expect(completed).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    expect(result.current.isPending).toBe(false);
    expect(result.current.error).toBeNull();
    window.removeEventListener("jobber:checkout-completed", completed);
  });

  it("surfaces a provider error and clears the baseline", async () => {
    const completed = vi.fn();
    window.addEventListener("jobber:checkout-completed", completed);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await act(async () => popupHandlers().onError());

    expect(completed).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    expect(result.current.isPending).toBe(false);
    await waitFor(() => expect(result.current.error).not.toBeNull());
  });

  it("retries after a failure instead of staying stuck", async () => {
    mockOpenPopup.mockRejectedValueOnce(new Error("script blocked"));
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await waitFor(() =>
      expect(result.current.error?.message).toBe("script blocked"),
    );
    expect(result.current.isPending).toBe(false);
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();

    await act(() => result.current.openCheckout("pro"));

    expect(mockOpenPopup).toHaveBeenCalledTimes(2);
    expect(result.current.error).toBeNull();
  });

  it("does not start a checkout for a plan the backend does not sell", async () => {
    mockConfig.value = { ...readyConfig, plans: ["pro"] };
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("enterprise"));

    expect(mockCreateSession).not.toHaveBeenCalled();
    expect(mockOpenPopup).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  it("does not start a checkout with no storefront to open it in", async () => {
    mockConfig.value = { ...readyConfig, storefront: "" };
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockCreateSession).not.toHaveBeenCalled();
    expect(mockOpenPopup).not.toHaveBeenCalled();
  });

  it("clears the baseline and surfaces the error when the session fails", async () => {
    mockCreateSession.mockRejectedValue(new Error("checkout unavailable"));
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockOpenPopup).not.toHaveBeenCalled();
    expect(assignSpy).not.toHaveBeenCalled();
    // A stale baseline would make the next page load poll for an upgrade that
    // can never arrive.
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    await waitFor(() =>
      expect(result.current.error?.message).toBe("checkout unavailable"),
    );
  });

  /**
   * A 204, a proxy that ate the body, a backend that answered `{}` — the
   * response is external data and the type says nothing about what actually
   * arrives. Reaching into it produced a raw `TypeError`, which is both an
   * internal detail and, worse, arrived *after* the popup call was skipped
   * with the busy state still being unwound by chance rather than by design.
   */
  it.each([
    ["an empty body", null],
    ["a body with no session", {}],
    ["a blank session id", { session_id: "" }],
  ])("refuses to open the popup on %s", async (_label, body) => {
    mockCreateSession.mockResolvedValue(body);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockOpenPopup).not.toHaveBeenCalled();
    expect(assignSpy).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    await waitFor(() =>
      expect(result.current.error?.message).toBe(
        "The checkout could not be completed",
      ),
    );
    // And the button is usable again, so a retry is one click away.
    expect(result.current.isPending).toBe(false);
  });

  it("lets a retry succeed after an unusable session response", async () => {
    mockCreateSession.mockResolvedValueOnce(null);
    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    mockCreateSession.mockResolvedValue({ session_id: "sess-2" });
    await act(() => result.current.openCheckout("pro"));

    expect(mockOpenPopup).toHaveBeenCalledOnce();
    expect(result.current.error).toBeNull();
  });

  /**
   * `sessionStorage.setItem` throws outright in Safari's private mode and on a
   * full quota. The baseline is a convenience for the *next* page load, so a
   * refusal to store it must not stop the purchase — and, when it sat outside
   * the try, the throw escaped `openCheckout` with the busy ref still set: the
   * button went dead for the rest of the page with nothing on screen to say
   * why.
   */
  describe("when sessionStorage refuses the baseline write", () => {
    function denyWrites(name: string) {
      return vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
        throw new DOMException(name, name);
      });
    }

    it.each(["QuotaExceededError", "SecurityError"])(
      "still opens the checkout on %s",
      async (name) => {
        const setItem = denyWrites(name);
        try {
          const { result } = renderHook(() => useCheckout());

          await act(() => result.current.openCheckout("pro"));

          expect(mockOpenPopup).toHaveBeenCalledTimes(1);
          expect(result.current.error).toBeNull();
        } finally {
          setItem.mockRestore();
        }
      },
    );

    it("leaves the busy state recoverable when the popup closes", async () => {
      const setItem = denyWrites("QuotaExceededError");
      try {
        const { result } = renderHook(() => useCheckout());
        await act(() => result.current.openCheckout("pro"));
        expect(result.current.isPending).toBe(true);

        act(() => popupHandlers().onClose({ completed: false }));

        expect(result.current.isPending).toBe(false);
      } finally {
        setItem.mockRestore();
      }
    });

    it("accepts a second click afterwards", async () => {
      const setItem = denyWrites("QuotaExceededError");
      try {
        const { result } = renderHook(() => useCheckout());
        await act(() => result.current.openCheckout("pro"));
        act(() => popupHandlers().onClose({ completed: false }));

        await act(() => result.current.openCheckout("pro"));

        expect(mockOpenPopup).toHaveBeenCalledTimes(2);
      } finally {
        setItem.mockRestore();
      }
    });
  });

  /**
   * Anything else thrown on the way to the popup — a cache read, a synchronous
   * provider failure — has to land in the terminal handler rather than escape
   * the hook: the customer needs the error, and the button needs to work again.
   */
  it("recovers and reports when the pre-popup work throws", async () => {
    mockGetQueryData.mockImplementation(() => {
      throw new Error("cache exploded");
    });
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    await waitFor(() =>
      expect(result.current.error?.message).toBe("cache exploded"),
    );
    expect(result.current.isPending).toBe(false);
    expect(mockOpenPopup).not.toHaveBeenCalled();

    mockGetQueryData.mockReturnValue({ plan: "free" });
    await act(() => result.current.openCheckout("pro"));

    expect(mockOpenPopup).toHaveBeenCalledTimes(1);
    expect(result.current.error).toBeNull();
  });
});

/**
 * The checkout popup covers the page and takes the keyboard with it. When it
 * goes, focus has to come back to the control the buyer opened it with — and by
 * the time the popup exists, the page can no longer work out which control that
 * was: starting a checkout disables the button, and a disabled button is
 * blurred. So the trigger is read here, synchronously, inside the click.
 */
describe("useCheckout — where focus goes when the popup closes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockConfig.value = readyConfig;
    mockCreateSession.mockResolvedValue({ session_id: "sess-1" });
    mockOpenPopup.mockResolvedValue(undefined);
    mockGetQueryData.mockReturnValue({ plan: "free" });
  });

  it("names the control that was focused when the checkout started", async () => {
    const trigger = document.createElement("button");
    document.body.appendChild(trigger);
    trigger.focus();

    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    const options = mockOpenPopup.mock.calls[0][0] as {
      returnFocusTo?: HTMLElement | null;
    };
    expect(options.returnFocusTo).toBe(trigger);

    trigger.remove();
  });

  it("names nothing rather than <body> when nothing was focused", async () => {
    document.body.focus();
    (document.activeElement as HTMLElement | null)?.blur();

    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    const options = mockOpenPopup.mock.calls[0][0] as {
      returnFocusTo?: HTMLElement | null;
    };
    // Not `document.body`: that is what activeElement reports for "nothing is
    // focused", and handing it to the overlay would have it "restore" the
    // keyboard to the top of the page instead of leaving the app to place it.
    expect(options.returnFocusTo).toBeNull();
  });
});

/**
 * A checkout that never opens has no overlay to release, and releasing the
 * overlay was the only thing that handed the keyboard back. The click itself
 * had already taken it away — the CTA goes busy, and a disabled button is
 * blurred by the browser — so a failed checkout left focus on `<body>`: an
 * error message the buyer could not tab to, at the top of a page they had
 * scrolled away from.
 *
 * The CTA being disabled and blurred while the request is in flight is what
 * these reproduce; it is not something the hook does, so the tests do it.
 */
describe("useCheckout — where focus goes when the checkout fails", () => {
  let trigger: HTMLButtonElement;

  beforeEach(() => {
    vi.clearAllMocks();
    mockConfig.value = readyConfig;
    mockCreateSession.mockResolvedValue({ session_id: "sess-1" });
    mockOpenPopup.mockResolvedValue(undefined);
    mockGetQueryData.mockReturnValue({ plan: "free" });

    trigger = document.createElement("button");
    document.body.appendChild(trigger);
    trigger.focus();
  });

  afterEach(() => {
    trigger.remove();
  });

  /**
   * What the browser does to the CTA once React puts it in its busy state.
   * Blurred first, then disabled — jsdom, like the platform, will not blur a
   * control that is already disabled.
   */
  function goBusy(): void {
    trigger.blur();
    trigger.disabled = true;
  }

  function comeBackFromBusy(): void {
    trigger.disabled = false;
  }

  it("hands the keyboard back when the session cannot be created", async () => {
    mockCreateSession.mockImplementation(async () => {
      goBusy();
      comeBackFromBusy();
      throw new Error("session refused");
    });

    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    expect(result.current.error).not.toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("hands the keyboard back when the popup cannot be opened", async () => {
    mockOpenPopup.mockImplementation(async () => {
      goBusy();
      comeBackFromBusy();
      throw new Error("script blocked");
    });

    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    expect(document.activeElement).toBe(trigger);
  });

  it("waits for the CTA to come out of its busy state", async () => {
    mockCreateSession.mockImplementation(async () => {
      goBusy();
      throw new Error("session refused");
    });

    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    // Still disabled: `focus()` on a disabled button does nothing at all, so
    // moving now would look correct and leave the buyer on `<body>`.
    expect(document.activeElement).not.toBe(trigger);

    await act(async () => {
      comeBackFromBusy();
    });

    await waitFor(() => expect(document.activeElement).toBe(trigger));
  });

  it("leaves focus alone when the buyer has moved on", async () => {
    const elsewhere = document.createElement("input");
    document.body.appendChild(elsewhere);

    mockCreateSession.mockImplementation(async () => {
      goBusy();
      comeBackFromBusy();
      elsewhere.focus();
      throw new Error("session refused");
    });

    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    expect(document.activeElement).toBe(elsewhere);
    elsewhere.remove();
  });

  it("moves nothing when no control started the checkout", async () => {
    trigger.blur();
    mockCreateSession.mockRejectedValue(new Error("session refused"));

    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    expect(result.current.error).not.toBeNull();
    // `<body>`, because nothing was focused to hand it back to — and body is
    // never a focus target this hook chooses.
    expect(document.activeElement).toBe(document.body);
  });

  it("hands the keyboard back when the provider reports an error", async () => {
    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    goBusy();
    comeBackFromBusy();
    act(() => popupHandlers().onError());

    expect(result.current.error).not.toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("gives up the wait when the hook unmounts", async () => {
    mockCreateSession.mockImplementation(async () => {
      goBusy();
      throw new Error("session refused");
    });

    const { result, unmount } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    unmount();
    await act(async () => {
      comeBackFromBusy();
    });

    // The control is focusable again, but the checkout it belonged to is gone.
    expect(document.activeElement).not.toBe(trigger);
  });
});
