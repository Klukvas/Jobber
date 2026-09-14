import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { AppLayout, POLL_INTERVAL_MS } from "../AppLayout";
import {
  PRE_CHECKOUT_PLAN_KEY,
  PRE_CHECKOUT_TTL_MS,
  notifyCheckoutCompleted,
  rememberPreCheckoutPlan,
} from "@/features/subscription/checkoutSignals";
import type { SubscriptionPlan } from "@/shared/types/api";

const plan = vi.hoisted(() => ({ value: "free" as SubscriptionPlan }));
const invalidateQueries = vi.hoisted(() => vi.fn());

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries }),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({ plan: plan.value }),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: unknown) => unknown) =>
    selector({ isAuthenticated: true }),
}));

vi.mock("@/features/onboarding/useOnboarding", () => ({
  useOnboarding: () => ({ shouldShow: false, complete: vi.fn() }),
}));

vi.mock("@/features/onboarding/WelcomeWizard", () => ({
  WelcomeWizard: () => null,
}));
vi.mock("@/widgets/Sidebar", () => ({ Sidebar: () => null }));
vi.mock("@/widgets/Header", () => ({ Header: () => null }));
vi.mock("@/features/support/SupportButton", () => ({
  SupportButton: () => null,
}));

vi.mock("@/features/subscription/components/SubscriptionSuccessModal", () => ({
  SubscriptionSuccessModal: ({ plan }: { plan: SubscriptionPlan | null }) =>
    plan ? <div data-testid="upgrade-success">{plan}</div> : null,
}));

/**
 * The router's idea of the query string.
 *
 * The layout's checkout-return handling has to read and rewrite *this*, not
 * `window.location.search`: under MemoryRouter — and under any client-side
 * navigation — the two disagree, and code reading the window would have been
 * silently untested here while quietly failing to clean the URL in the app.
 */
function LocationSearch() {
  return <div data-testid="location-search">{useLocation().search}</div>;
}

function renderLayout(search = "") {
  return render(
    <MemoryRouter initialEntries={[`/app/jobs${search}`]}>
      <AppLayout />
      <LocationSearch />
    </MemoryRouter>,
  );
}

function locationSearch(): string {
  return screen.getByTestId("location-search").textContent ?? "";
}

describe("AppLayout — returning from checkout", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    plan.value = "free";
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  it("shows no success modal and does not poll without a pending checkout", () => {
    renderLayout();

    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
    expect(invalidateQueries).not.toHaveBeenCalled();
  });

  it("polls after a checkout without showing success while the plan is unchanged", async () => {
    // A reload while the popup was open: the baseline is still on disk, but
    // the backend keeps reporting the free plan.
    rememberPreCheckoutPlan("free");

    renderLayout();

    await waitFor(() =>
      expect(invalidateQueries).toHaveBeenCalledWith({
        queryKey: ["subscription"],
      }),
    );
    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
  });

  it("shows success once the backend reports a higher plan", async () => {
    rememberPreCheckoutPlan("free");
    plan.value = "pro";

    renderLayout();

    expect(await screen.findByTestId("upgrade-success")).toHaveTextContent(
      "pro",
    );
  });

  it("does not celebrate a downgrade or a sideways move", () => {
    rememberPreCheckoutPlan("enterprise");
    plan.value = "pro";

    renderLayout();

    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
  });

  it("clears the baseline so a later visit does not re-poll", async () => {
    rememberPreCheckoutPlan("free");

    renderLayout();

    await waitFor(() =>
      expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull(),
    );
  });

  it("still works when the storefront redirects back with the success param", async () => {
    // The FastSpring storefront's post-order redirect is a dashboard setting we
    // do not control, so both entry paths must behave identically.
    rememberPreCheckoutPlan("free");
    plan.value = "pro";

    renderLayout("?subscription=success");

    expect(await screen.findByTestId("upgrade-success")).toHaveTextContent(
      "pro",
    );
  });
});

/**
 * The `?subscription=success` parameter is the storefront's, appended by a
 * dashboard redirect setting. It is a breadcrumb, never evidence — anyone can
 * bookmark it, share it, or land on it twice — so the only thing it is allowed
 * to do is get itself cleaned out of the address bar.
 */
describe("AppLayout — the subscription URL parameter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    plan.value = "free";
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  it("removes itself from the router's URL after a checkout return", async () => {
    rememberPreCheckoutPlan("free");

    renderLayout("?subscription=success");

    await waitFor(() => expect(locationSearch()).not.toContain("subscription"));
  });

  it("keeps every other parameter on the page", async () => {
    rememberPreCheckoutPlan("free");

    renderLayout("?tab=billing&subscription=success&highlight=pro");

    await waitFor(() => expect(locationSearch()).not.toContain("subscription"));
    const params = new URLSearchParams(locationSearch());
    expect(params.get("tab")).toBe("billing");
    expect(params.get("highlight")).toBe("pro");
  });

  it("leaves a page that never came back from checkout untouched", () => {
    renderLayout("?tab=billing");

    expect(locationSearch()).toBe("?tab=billing");
  });

  // A bookmarked success URL, opened by somebody who is already a subscriber:
  // no checkout happened, so there is no baseline, and "free -> pro" is just
  // the plan they already had. Celebrating it claims a purchase that was
  // never made.
  it("never celebrates without a baseline the app itself armed", async () => {
    plan.value = "pro";

    renderLayout("?subscription=success");

    await waitFor(() => expect(locationSearch()).not.toContain("subscription"));
    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(invalidateQueries).not.toHaveBeenCalled();
  });

  it("does not block a bookmarked success URL behind the overlay", () => {
    renderLayout("?subscription=success");

    expect(
      screen.queryByText("settings.subscription.activating"),
    ).not.toBeInTheDocument();
  });

  // An expired baseline is no baseline: the parameter must not resurrect it.
  it("ignores the parameter when the baseline has expired", () => {
    vi.useFakeTimers();
    try {
      rememberPreCheckoutPlan("free");
      vi.advanceTimersByTime(PRE_CHECKOUT_TTL_MS + 1);
      plan.value = "pro";

      renderLayout("?subscription=success");

      expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("AppLayout — the activating overlay is escapable", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    plan.value = "free";
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  it("shows the overlay while a checkout return is still pending", () => {
    rememberPreCheckoutPlan("free");

    renderLayout();

    expect(
      screen.getByText("settings.subscription.activating"),
    ).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toHaveAttribute("aria-modal", "true");
  });

  it("puts focus on the escape hatch, the overlay's only control", () => {
    rememberPreCheckoutPlan("free");

    renderLayout();

    expect(
      screen.getByText("settings.subscription.activatingDismiss"),
    ).toHaveFocus();
  });

  it("closes the overlay the moment the user asks to continue", async () => {
    // The buyer reloaded mid-purchase and then gave up. Without this button
    // they would sit behind a spinner until the five-minute poll expired.
    rememberPreCheckoutPlan("free");

    renderLayout();

    fireEvent.click(
      screen.getByText("settings.subscription.activatingDismiss"),
    );

    expect(
      screen.queryByText("settings.subscription.activating"),
    ).not.toBeInTheDocument();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
  });

  it("does not reopen the overlay after it was dismissed", () => {
    // Fake timers on purpose. The poll only fires every POLL_INTERVAL_MS, so
    // under real timers "no further calls arrived" was true the instant it was
    // asserted and would have stayed true with the teardown deleted entirely.
    vi.useFakeTimers();
    try {
      rememberPreCheckoutPlan("free");

      renderLayout();
      const callsWhilePolling = invalidateQueries.mock.calls.length;
      expect(callsWhilePolling).toBeGreaterThan(0);

      fireEvent.click(
        screen.getByText("settings.subscription.activatingDismiss"),
      );

      const callsAfterDismiss = invalidateQueries.mock.calls.length;
      act(() => {
        vi.advanceTimersByTime(POLL_INTERVAL_MS * 5);
      });

      expect(invalidateQueries.mock.calls.length).toBe(callsAfterDismiss);
      expect(
        screen.queryByText("settings.subscription.activating"),
      ).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  // The interval is what keeps the overlay honest: it is the only thing that
  // can notice the backend granting the plan. A poll that never fired would
  // leave a paying customer staring at the spinner.
  it("keeps polling on the interval while it is still waiting", () => {
    vi.useFakeTimers();
    try {
      rememberPreCheckoutPlan("free");

      renderLayout();
      const initialCalls = invalidateQueries.mock.calls.length;

      act(() => {
        vi.advanceTimersByTime(POLL_INTERVAL_MS * 3);
      });

      expect(invalidateQueries.mock.calls.length).toBe(initialCalls + 3);
    } finally {
      vi.useRealTimers();
    }
  });

  it("dismissing never celebrates a purchase that did not happen", () => {
    rememberPreCheckoutPlan("pro");
    plan.value = "pro";

    renderLayout();
    fireEvent.click(
      screen.getByText("settings.subscription.activatingDismiss"),
    );

    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
  });

  it("clears the baseline when the page is restored from the bfcache", () => {
    // Back from the Account Management Portal can restore this page without a
    // remount, so nothing else clears the baseline and the next full load would
    // poll for a purchase that never happened.
    rememberPreCheckoutPlan("free");

    renderLayout();
    expect(
      screen.getByText("settings.subscription.activating"),
    ).toBeInTheDocument();

    act(() => {
      const restored = new Event("pageshow");
      Object.defineProperty(restored, "persisted", { value: true });
      window.dispatchEvent(restored);
    });

    expect(
      screen.queryByText("settings.subscription.activating"),
    ).not.toBeInTheDocument();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  it("a plain page load is not treated as a bfcache restore", () => {
    rememberPreCheckoutPlan("free");

    renderLayout();

    act(() => {
      const fresh = new Event("pageshow");
      Object.defineProperty(fresh, "persisted", { value: false });
      window.dispatchEvent(fresh);
    });

    expect(
      screen.getByText("settings.subscription.activating"),
    ).toBeInTheDocument();
  });
});

describe("AppLayout — the popup closes without navigating", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    plan.value = "free";
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  it("starts watching when the popup reports an order on this page load", async () => {
    // Nothing pending at mount: the purchase happens entirely on this page, so
    // there is no navigation for the mount-time read to notice.
    renderLayout();
    expect(
      screen.queryByText("settings.subscription.activating"),
    ).not.toBeInTheDocument();

    rememberPreCheckoutPlan("free");
    act(() => notifyCheckoutCompleted());

    expect(
      screen.getByText("settings.subscription.activating"),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(invalidateQueries).toHaveBeenCalledWith({
        queryKey: ["subscription"],
      }),
    );
    // The provider's callback is not payment: nothing is granted yet.
    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
  });

  it("celebrates only once the backend reports the higher plan", async () => {
    plan.value = "pro";
    renderLayout();
    // The backend already says pro, but no checkout is pending, so this is just
    // an existing subscriber loading the app.
    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();

    rememberPreCheckoutPlan("free");
    act(() => notifyCheckoutCompleted());

    expect(await screen.findByTestId("upgrade-success")).toHaveTextContent(
      "pro",
    );
  });

  it("lifts the baseline off disk so the next page load stays quiet", async () => {
    renderLayout();

    rememberPreCheckoutPlan("free");
    act(() => notifyCheckoutCompleted());

    await waitFor(() =>
      expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull(),
    );
  });
});

/**
 * The provider does not always tell us the popup closed — its own X fires no
 * callback — so a baseline can outlive the checkout that wrote it. A page load
 * must not turn that leftover into a blocking "Activating your subscription…"
 * for someone who declined to pay.
 */
describe("AppLayout — a stale checkout baseline", () => {
  beforeEach(() => {
    sessionStorage.clear();
    plan.value = "free";
    vi.useRealTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not show the overlay for a baseline older than the window", () => {
    vi.useFakeTimers();
    rememberPreCheckoutPlan("free");
    vi.advanceTimersByTime(PRE_CHECKOUT_TTL_MS + 1);

    renderLayout();

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("still shows the overlay for a checkout that just started", () => {
    rememberPreCheckoutPlan("free");

    renderLayout();

    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("clears the expired key so later navigations stay quiet", () => {
    vi.useFakeTimers();
    rememberPreCheckoutPlan("free");
    vi.advanceTimersByTime(PRE_CHECKOUT_TTL_MS + 1);

    renderLayout();

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });
});
