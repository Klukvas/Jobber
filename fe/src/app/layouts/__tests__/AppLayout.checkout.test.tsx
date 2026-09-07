import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { AppLayout } from "../AppLayout";
import {
  PRE_CHECKOUT_PLAN_KEY,
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

function renderLayout(search = "") {
  return render(
    <MemoryRouter initialEntries={[`/app/jobs${search}`]}>
      <AppLayout />
    </MemoryRouter>,
  );
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
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

    renderLayout();

    await waitFor(() =>
      expect(invalidateQueries).toHaveBeenCalledWith({
        queryKey: ["subscription"],
      }),
    );
    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
  });

  it("shows success once the backend reports a higher plan", async () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");
    plan.value = "pro";

    renderLayout();

    expect(await screen.findByTestId("upgrade-success")).toHaveTextContent(
      "pro",
    );
  });

  it("does not celebrate a downgrade or a sideways move", () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "enterprise");
    plan.value = "pro";

    renderLayout();

    expect(screen.queryByTestId("upgrade-success")).not.toBeInTheDocument();
  });

  it("clears the baseline so a later visit does not re-poll", async () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

    renderLayout();

    await waitFor(() =>
      expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull(),
    );
  });

  it("still works when the storefront redirects back with the success param", async () => {
    // The FastSpring storefront's post-order redirect is a dashboard setting we
    // do not control, so both entry paths must behave identically.
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");
    plan.value = "pro";

    renderLayout("?subscription=success");

    expect(await screen.findByTestId("upgrade-success")).toHaveTextContent(
      "pro",
    );
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
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

    renderLayout();

    expect(
      screen.getByText("settings.subscription.activating"),
    ).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toHaveAttribute("aria-modal", "true");
  });

  it("puts focus on the escape hatch, the overlay's only control", () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

    renderLayout();

    expect(
      screen.getByText("settings.subscription.activatingDismiss"),
    ).toHaveFocus();
  });

  it("closes the overlay the moment the user asks to continue", async () => {
    // The buyer reloaded mid-purchase and then gave up. Without this button
    // they would sit behind a spinner until the five-minute poll expired.
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

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

  it("does not reopen the overlay after it was dismissed", async () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

    renderLayout();
    fireEvent.click(
      screen.getByText("settings.subscription.activatingDismiss"),
    );

    // Polling has stopped, so nothing can put the overlay back.
    const callsAfterDismiss = invalidateQueries.mock.calls.length;
    await waitFor(() =>
      expect(
        screen.queryByText("settings.subscription.activating"),
      ).not.toBeInTheDocument(),
    );
    expect(invalidateQueries.mock.calls.length).toBe(callsAfterDismiss);
  });

  it("dismissing never celebrates a purchase that did not happen", () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "pro");
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
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

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
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "free");

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
