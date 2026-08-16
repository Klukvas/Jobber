import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ScheduleStageModal } from "../ScheduleStageModal";

const mockGetStatus = vi.hoisted(() => vi.fn());
const mockGetAuthURL = vi.hoisted(() => vi.fn());
const mockCreateEvent = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("@/services/calendarService", () => ({
  calendarService: {
    getStatus: mockGetStatus,
    getAuthURL: mockGetAuthURL,
    createEvent: mockCreateEvent,
  },
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

function renderModal(overrides?: {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const onOpenChange = overrides?.onOpenChange ?? vi.fn();
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <ScheduleStageModal
        open={overrides?.open ?? true}
        onOpenChange={onOpenChange}
        stageId="stage-1"
        stageName="Phone Screen"
        jobId="job-1"
      />
    </QueryClientProvider>,
  );
  return { ...utils, onOpenChange };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ScheduleStageModal — calendar not connected", () => {
  beforeEach(() => {
    mockGetStatus.mockResolvedValue({ connected: false });
  });

  it("shows the connect-calendar empty state", async () => {
    renderModal();
    expect(
      await screen.findByText("jobs.schedule.calendarNotConnected"),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "settings.calendar.connect" }),
    ).toBeInTheDocument();
  });

  it("redirects to the auth URL when connect succeeds", async () => {
    const user = userEvent.setup();
    mockGetAuthURL.mockResolvedValue({ url: "https://oauth.test/go" });
    // Replace window.location (configurable at the window level) so the connect
    // success handler's `window.location.href = url` assignment is observable.
    const originalLocation = window.location;
    Object.defineProperty(window, "location", {
      configurable: true,
      writable: true,
      value: { href: "" } as unknown as Location,
    });

    renderModal();
    await screen.findByText("jobs.schedule.calendarNotConnected");
    await user.click(
      screen.getByRole("button", { name: "settings.calendar.connect" }),
    );

    await waitFor(() =>
      expect(window.location.href).toBe("https://oauth.test/go"),
    );

    Object.defineProperty(window, "location", {
      configurable: true,
      writable: true,
      value: originalLocation,
    });
  });

  it("shows an error when fetching the auth URL fails", async () => {
    const user = userEvent.setup();
    mockGetAuthURL.mockRejectedValue(new Error("oauth boom"));
    renderModal();
    await screen.findByText("jobs.schedule.calendarNotConnected");
    await user.click(
      screen.getByRole("button", { name: "settings.calendar.connect" }),
    );

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("oauth boom"),
    );
  });
});

describe("ScheduleStageModal — calendar connected", () => {
  beforeEach(() => {
    mockGetStatus.mockResolvedValue({ connected: true });
  });

  it("shows the scheduling form prefilled with the stage name", async () => {
    renderModal();
    const titleInput = await screen.findByLabelText(
      /jobs\.schedule\.eventTitle/,
    );
    expect(titleInput).toHaveValue("Phone Screen");
    // submit disabled until a start time is chosen
    expect(
      screen.getByRole("button", { name: "jobs.schedule.schedule" }),
    ).toBeDisabled();
  });

  it("creates a calendar event with an ISO start time and duration", async () => {
    const onOpenChange = vi.fn();
    mockCreateEvent.mockResolvedValue({ id: "evt-1" });
    const { container } = renderModal({ onOpenChange });

    await screen.findByLabelText(/jobs\.schedule\.eventTitle/);
    fireEvent.change(screen.getByLabelText(/jobs\.schedule\.startTime/), {
      target: { value: "2026-03-01T10:00" },
    });
    // change duration to 90 minutes
    fireEvent.change(screen.getByLabelText(/jobs\.schedule\.duration/), {
      target: { value: "90" },
    });

    fireEvent.submit(container.querySelector("form")!);

    await waitFor(() => expect(mockCreateEvent).toHaveBeenCalledTimes(1));
    const payload = mockCreateEvent.mock.calls[0][0];
    expect(payload.stage_id).toBe("stage-1");
    expect(payload.title).toBe("Phone Screen");
    expect(payload.duration_min).toBe(90);
    // start_time carries the local datetime plus a timezone offset suffix
    expect(payload.start_time).toMatch(/^2026-03-01T10:00:00[+-]\d{2}:\d{2}$/);
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith(
        "jobs.schedule.createSuccess",
      ),
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("does not submit when the start time is empty", async () => {
    const { container } = renderModal();
    await screen.findByLabelText(/jobs\.schedule\.eventTitle/);

    fireEvent.submit(container.querySelector("form")!);

    expect(mockCreateEvent).not.toHaveBeenCalled();
  });

  it("surfaces a create-event error", async () => {
    mockCreateEvent.mockRejectedValue(new Error("event boom"));
    const { container } = renderModal();

    await screen.findByLabelText(/jobs\.schedule\.eventTitle/);
    fireEvent.change(screen.getByLabelText(/jobs\.schedule\.startTime/), {
      target: { value: "2026-03-01T10:00" },
    });
    fireEvent.submit(container.querySelector("form")!);

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("event boom"),
    );
  });
});
