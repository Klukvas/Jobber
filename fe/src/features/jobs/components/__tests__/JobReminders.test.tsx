import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { JobReminders } from "../JobReminders";
import type { ReminderDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockMutationState = vi.hoisted(() => ({ isPending: false }));
const mockQueries = vi.hoisted(() => ({
  reminders: [] as ReminderDTO[],
  isLoading: false,
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: mockQueries.reminders,
    isLoading: mockQueries.isLoading,
  }),
  useMutation: (opts: {
    mutationFn: (v: unknown) => Promise<unknown>;
    onSuccess?: () => void;
    onError?: (e: Error) => void;
  }) => ({
    mutate: (v: unknown) =>
      Promise.resolve(opts.mutationFn(v)).then(opts.onSuccess, opts.onError),
    isPending: mockMutationState.isPending,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

const mockRemindersService = vi.hoisted(() => ({
  listByJob: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
}));
vi.mock("@/services/remindersService", () => ({
  remindersService: mockRemindersService,
}));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

function reminder(overrides: Partial<ReminderDTO> = {}): ReminderDTO {
  return {
    id: "r1",
    job_id: "job-1",
    message: "Follow up",
    remind_at: "2026-09-01T12:00:00Z",
    is_done: false,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  } as ReminderDTO;
}

describe("JobReminders", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
    mockQueries.reminders = [];
    mockQueries.isLoading = false;
  });

  it("shows an empty-state message when there are no reminders", () => {
    render(<JobReminders jobId="job-1" />);
    expect(screen.getByText("jobs.noReminders")).toBeInTheDocument();
  });

  it("does not show the empty state while loading", () => {
    mockQueries.isLoading = true;
    render(<JobReminders jobId="job-1" />);
    expect(screen.queryByText("jobs.noReminders")).not.toBeInTheDocument();
  });

  it("renders reminders and toggles completion", async () => {
    const user = userEvent.setup();
    mockQueries.reminders = [reminder()];
    mockRemindersService.update.mockResolvedValue({});

    render(<JobReminders jobId="job-1" />);
    expect(screen.getByText("Follow up")).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox"));
    expect(mockRemindersService.update).toHaveBeenCalledWith("r1", {
      is_done: true,
    });
  });

  it("deletes a reminder on trash click", async () => {
    const user = userEvent.setup();
    mockQueries.reminders = [reminder()];
    mockRemindersService.delete.mockResolvedValue(undefined);

    render(<JobReminders jobId="job-1" />);
    await user.click(screen.getByRole("button", { name: "common.delete" }));
    expect(mockRemindersService.delete).toHaveBeenCalledWith("r1");
  });

  it("keeps the add button disabled until message and time are provided", async () => {
    const user = userEvent.setup();
    render(<JobReminders jobId="job-1" />);

    const add = screen.getByRole("button", { name: "jobs.addReminder" });
    expect(add).toBeDisabled();

    await user.type(
      screen.getByPlaceholderText("jobs.reminderPlaceholder"),
      "Call recruiter",
    );
    // Still disabled without a time.
    expect(add).toBeDisabled();
  });

  it("creates a reminder with a UTC ISO remind_at", async () => {
    const user = userEvent.setup();
    mockRemindersService.create.mockResolvedValue({});

    render(<JobReminders jobId="job-1" />);
    await user.type(
      screen.getByPlaceholderText("jobs.reminderPlaceholder"),
      "Call recruiter",
    );

    const dateInput = screen.getByLabelText("jobs.reminderTime");
    await user.type(dateInput, "2026-09-01T09:30");
    await user.click(screen.getByRole("button", { name: "jobs.addReminder" }));

    await waitFor(() => expect(mockRemindersService.create).toHaveBeenCalled());
    const arg = mockRemindersService.create.mock.calls[0][0];
    expect(arg.job_id).toBe("job-1");
    expect(arg.message).toBe("Call recruiter");
    // Converted to a UTC ISO string.
    expect(arg.remind_at).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*Z$/);
    expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
      "jobs.reminderAddedSuccess",
    );
  });
});
