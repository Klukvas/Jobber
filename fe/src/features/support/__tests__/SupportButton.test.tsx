import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SupportButton } from "../SupportButton";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars?.count !== undefined ? `${key}:${vars.count}` : key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useLocation: () => ({ pathname: "/app/jobs" }),
}));

const mockMutate = vi.hoisted(() => vi.fn());
const mockMutationState = vi.hoisted(() => ({ isPending: false }));

vi.mock("@tanstack/react-query", () => ({
  useMutation: (opts: {
    mutationFn: (v: unknown) => Promise<unknown>;
    onSuccess?: () => void;
    onError?: (e: Error) => void;
  }) => ({
    mutate: (v: unknown) => {
      mockMutate(v);
      return Promise.resolve(opts.mutationFn(v)).then(
        opts.onSuccess,
        opts.onError,
      );
    },
    isPending: mockMutationState.isPending,
  }),
}));

const mockSupportService = vi.hoisted(() => ({ submit: vi.fn() }));
vi.mock("@/services/supportService", () => ({
  supportService: mockSupportService,
}));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

async function openDialog(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "support.title" }));
  await screen.findByText("support.description");
}

describe("SupportButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
  });

  it("does not render the dialog until the trigger is clicked", () => {
    render(<SupportButton />);
    expect(screen.queryByText("support.description")).not.toBeInTheDocument();
  });

  it("opens the support dialog on click", async () => {
    const user = userEvent.setup();
    render(<SupportButton />);
    await openDialog(user);
    expect(screen.getByText("support.title")).toBeInTheDocument();
  });

  it("disables send until subject and message meet the minimum length", async () => {
    const user = userEvent.setup();
    render(<SupportButton />);
    await openDialog(user);

    const send = screen.getByRole("button", { name: "support.send" });
    expect(send).toBeDisabled();

    await user.type(screen.getByLabelText("support.subject"), "Hi");
    // Subject still too short (min 3) -> hint shown, still disabled.
    expect(send).toBeDisabled();

    await user.type(screen.getByLabelText("support.subject"), "!");
    await user.type(
      screen.getByLabelText("support.message"),
      "This is long enough",
    );
    expect(send).toBeEnabled();
  });

  it("submits the trimmed values with the current page and shows success", async () => {
    const user = userEvent.setup();
    mockSupportService.submit.mockResolvedValue({ message: "ok" });
    render(<SupportButton />);
    await openDialog(user);

    await user.type(screen.getByLabelText("support.subject"), "  Need help  ");
    await user.type(
      screen.getByLabelText("support.message"),
      "  My detailed problem here  ",
    );
    await user.click(screen.getByRole("button", { name: "support.send" }));

    await waitFor(() =>
      expect(mockSupportService.submit).toHaveBeenCalledWith({
        subject: "Need help",
        message: "My detailed problem here",
        page: "/app/jobs",
      }),
    );
    expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
      "support.success",
    );
  });

  it("surfaces the error message when submission fails", async () => {
    const user = userEvent.setup();
    mockSupportService.submit.mockRejectedValue(new Error("rate limited"));
    render(<SupportButton />);
    await openDialog(user);

    await user.type(screen.getByLabelText("support.subject"), "Need help");
    await user.type(
      screen.getByLabelText("support.message"),
      "My detailed problem here",
    );
    await user.click(screen.getByRole("button", { name: "support.send" }));

    await waitFor(() =>
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "rate limited",
      ),
    );
  });
});
