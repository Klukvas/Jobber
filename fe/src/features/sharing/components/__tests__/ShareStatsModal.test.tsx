import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/services/api";
import type { ShareDTO } from "@/services/sharingService";
import type {
  OverviewAnalytics,
  FunnelAnalytics,
} from "@/services/analyticsService";
import { ShareStatsModal } from "../ShareStatsModal";

// --- hoisted service mocks (real ApiError so instanceof works) ---
const mockList = vi.hoisted(() => vi.fn());
const mockCreate = vi.hoisted(() => vi.fn());
const mockRemove = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("@/services/sharingService", () => ({
  sharingService: {
    list: mockList,
    create: mockCreate,
    remove: mockRemove,
  },
  buildShareUrl: (token: string) => `https://app.test/s/${token}`,
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: mockShowSuccess,
  showErrorNotification: mockShowError,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

// --- factories ---
function makeOverview(
  overrides?: Partial<OverviewAnalytics>,
): OverviewAnalytics {
  return {
    total_applications: 42,
    active_applications: 10,
    closed_applications: 32,
    rejected_applications: 20,
    response_rate: 25.5,
    avg_days_to_first_response: 5,
    ...overrides,
  };
}

function makeFunnel(): FunnelAnalytics {
  return {
    stages: [
      {
        stage_name: "Applied",
        stage_order: 1,
        count: 42,
        conversion_rate: 100,
        drop_off_rate: 0,
      },
      {
        stage_name: "Interview",
        stage_order: 2,
        count: 12,
        conversion_rate: 28.5,
        drop_off_rate: 71.5,
      },
    ],
  };
}

function makeShare(overrides?: Partial<ShareDTO>): ShareDTO {
  return {
    id: "share-1",
    token: "tok-abc",
    created_at: "2026-01-15T00:00:00Z",
    snapshot: {
      schema_version: 1,
      generated_at: "2026-01-15T00:00:00Z",
      overview: makeOverview(),
      funnel: [],
    },
    ...overrides,
  };
}

function renderRaw(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

function renderModal(props?: {
  overview?: OverviewAnalytics;
  funnel?: FunnelAnalytics;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}) {
  const onOpenChange = props?.onOpenChange ?? vi.fn();
  const utils = renderRaw(
    <ShareStatsModal
      open={props?.open ?? true}
      onOpenChange={onOpenChange}
      overview={props?.overview ?? makeOverview()}
      funnel={props?.funnel ?? makeFunnel()}
    />,
  );
  return { ...utils, onOpenChange };
}

const clipboardWrite = vi.fn().mockResolvedValue(undefined);

// user-event's setup() installs its own clipboard stub, so the mock must be
// (re)installed AFTER setup() to observe the component's writeText calls.
function installClipboard() {
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: clipboardWrite },
    configurable: true,
    writable: true,
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  mockList.mockResolvedValue([]);
  clipboardWrite.mockResolvedValue(undefined);
  installClipboard();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("ShareStatsModal", () => {
  it("does not render when closed", () => {
    const { container } = renderModal({ open: false });
    expect(container.innerHTML).toBe("");
  });

  it("renders title, description and the consent preview when open", async () => {
    renderModal();
    expect(await screen.findByText("sharing.modalTitle")).toBeInTheDocument();
    expect(screen.getByText("sharing.modalDescription")).toBeInTheDocument();
    expect(screen.getByText("sharing.whatWillBeShared")).toBeInTheDocument();
    expect(screen.getByText("sharing.privacyNote")).toBeInTheDocument();
  });

  it("shows the overview totals and funnel stage names in the preview", () => {
    renderModal({ overview: makeOverview({ total_applications: 99 }) });
    // total_applications rendered
    expect(screen.getByText(/99/)).toBeInTheDocument();
    // response_rate rendered with % suffix
    expect(screen.getByText(/25\.5%/)).toBeInTheDocument();
    // funnel stage names joined
    expect(screen.getByText(/Applied → Interview/)).toBeInTheDocument();
  });

  it("shows placeholders when overview and funnel are missing", () => {
    renderRaw(
      <ShareStatsModal
        open={true}
        onOpenChange={vi.fn()}
        overview={undefined}
        funnel={undefined}
      />,
    );
    // The consent list renders "—" placeholders for each missing metric.
    const listItems = screen
      .getAllByRole("listitem")
      .filter((li) => li.textContent?.includes("—"));
    expect(listItems.length).toBeGreaterThanOrEqual(3);
  });

  it("disables the create button when there is no overview", () => {
    renderRaw(
      <ShareStatsModal
        open={true}
        onOpenChange={vi.fn()}
        overview={undefined}
        funnel={undefined}
      />,
    );
    const createBtn = screen.getByRole("button", {
      name: "sharing.createButton",
    });
    expect(createBtn).toBeDisabled();
  });

  it("shows an empty message when there are no existing shares", async () => {
    mockList.mockResolvedValue([]);
    renderModal();
    expect(await screen.findByText("sharing.noShares")).toBeInTheDocument();
  });

  it("lists existing shares returned by the service", async () => {
    mockList.mockResolvedValue([
      makeShare({ id: "s1", token: "tok-1" }),
      makeShare({ id: "s2", token: "tok-2" }),
    ]);
    renderModal();
    expect(
      await screen.findByText("https://app.test/s/tok-1"),
    ).toBeInTheDocument();
    expect(screen.getByText("https://app.test/s/tok-2")).toBeInTheDocument();
  });

  it("does not query shares while closed (enabled=open)", () => {
    renderModal({ open: false });
    expect(mockList).not.toHaveBeenCalled();
  });

  it("creates a share, shows the ready link and copies it on success", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockCreate.mockResolvedValue(makeShare({ token: "new-tok" }));
    renderModal();

    await user.click(
      screen.getByRole("button", { name: "sharing.createButton" }),
    );

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    // "link ready" panel appears with the generated URL
    expect(await screen.findByText("sharing.linkReady")).toBeInTheDocument();
    expect(screen.getByText("https://app.test/s/new-tok")).toBeInTheDocument();
    // auto-copy fired
    await waitFor(() =>
      expect(clipboardWrite).toHaveBeenCalledWith("https://app.test/s/new-tok"),
    );
    expect(mockShowSuccess).toHaveBeenCalledWith("sharing.linkCopied");
  });

  it("maps SHARE_LIMIT_REACHED to a friendly error", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockCreate.mockRejectedValue(
      new ApiError("nope", "SHARE_LIMIT_REACHED", 403),
    );
    renderModal();

    await user.click(
      screen.getByRole("button", { name: "sharing.createButton" }),
    );

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("sharing.limitReached"),
    );
  });

  it("falls back to the error message on generic create failure", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockCreate.mockRejectedValue(new Error("boom"));
    renderModal();

    await user.click(
      screen.getByRole("button", { name: "sharing.createButton" }),
    );

    await waitFor(() => expect(mockShowError).toHaveBeenCalledWith("boom"));
  });

  it("copies an existing share link via its copy button", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockList.mockResolvedValue([makeShare({ id: "s1", token: "copy-me" })]);
    renderModal();

    const url = await screen.findByText("https://app.test/s/copy-me");
    const row = url.closest("li") as HTMLLIElement;
    // The share row has two buttons: copy (first) and revoke (second).
    const rowButtons = within(row).getAllByRole("button");
    await user.click(rowButtons[0]);

    await waitFor(() =>
      expect(clipboardWrite).toHaveBeenCalledWith("https://app.test/s/copy-me"),
    );
    expect(mockShowSuccess).toHaveBeenCalledWith("sharing.linkCopied");
  });

  it("revokes a share and shows a success toast", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockList.mockResolvedValue([makeShare({ id: "s1", token: "del-me" })]);
    mockRemove.mockResolvedValue(undefined);
    renderModal();

    const url = await screen.findByText("https://app.test/s/del-me");
    const row = url.closest("li") as HTMLLIElement;
    const rowButtons = within(row).getAllByRole("button");
    // revoke (trash) button is the second button in the share row
    await user.click(rowButtons[1]);

    await waitFor(() => expect(mockRemove).toHaveBeenCalledWith("s1"));
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("sharing.revoked"),
    );
  });

  it("surfaces a revoke error via the error notification", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockList.mockResolvedValue([makeShare({ id: "s1", token: "del-me" })]);
    mockRemove.mockRejectedValue(new Error("cannot revoke"));
    renderModal();

    const url = await screen.findByText("https://app.test/s/del-me");
    const row = url.closest("li") as HTMLLIElement;
    const rowButtons = within(row).getAllByRole("button");
    await user.click(rowButtons[1]);

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("cannot revoke"),
    );
  });

  it("shows an error toast if clipboard write rejects during copy", async () => {
    const user = userEvent.setup();
    installClipboard();
    clipboardWrite.mockRejectedValue(new Error("denied"));
    mockList.mockResolvedValue([makeShare({ id: "s1", token: "tok-1" })]);
    renderModal();

    const url = await screen.findByText("https://app.test/s/tok-1");
    const row = url.closest("li") as HTMLLIElement;
    const rowButtons = within(row).getAllByRole("button");
    await user.click(rowButtons[0]);

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("sharing.copyFailed"),
    );
  });

  it("resets the created link when the modal is closed", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockCreate.mockResolvedValue(makeShare({ token: "new-tok" }));
    const onOpenChange = vi.fn();
    renderModal({ onOpenChange });

    await user.click(
      screen.getByRole("button", { name: "sharing.createButton" }),
    );
    await screen.findByText("sharing.linkReady");

    // Close via the X button
    await user.click(screen.getByRole("button", { name: "common.close" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
