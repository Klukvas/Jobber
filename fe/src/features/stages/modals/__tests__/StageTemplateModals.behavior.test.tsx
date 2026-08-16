import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/services/api";
import type { StageTemplateDTO } from "@/shared/types/api";
import { CreateStageTemplateModal } from "../CreateStageTemplateModal";
import { EditStageTemplateModal } from "../EditStageTemplateModal";

const mockCreate = vi.hoisted(() => vi.fn());
const mockUpdate = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    create: mockCreate,
    update: mockUpdate,
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

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

const template: StageTemplateDTO = {
  id: "st-1",
  name: "Phone Screen",
  order: 2,
  created_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("CreateStageTemplateModal — behavior", () => {
  it("preselects the order passed via initialOrder", () => {
    renderWithClient(
      <CreateStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        initialOrder={5}
      />,
    );
    expect(screen.getByLabelText(/stages\.order/)).toHaveValue(5);
  });

  it("keeps the submit button disabled until name and order are filled", async () => {
    const user = userEvent.setup();
    renderWithClient(
      <CreateStageTemplateModal open={true} onOpenChange={vi.fn()} />,
    );
    const submit = screen.getByRole("button", { name: "common.create" });
    expect(submit).toBeDisabled();

    await user.type(screen.getByLabelText(/stages\.stageName/), "Onsite");
    // order still empty -> disabled
    expect(submit).toBeDisabled();

    await user.type(screen.getByLabelText(/stages\.order/), "3");
    expect(submit).toBeEnabled();
  });

  it("submits create with parsed order and shows success", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockCreate.mockResolvedValue({ ...template, id: "new" });
    renderWithClient(
      <CreateStageTemplateModal open={true} onOpenChange={onOpenChange} />,
    );

    await user.type(screen.getByLabelText(/stages\.stageName/), "Onsite");
    await user.type(screen.getByLabelText(/stages\.order/), "4");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    // React Query passes a mutation context as the 2nd arg to a bare
    // mutationFn reference — assert on the first (variables) arg only.
    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({ name: "Onsite", order: 4 });
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("stages.createSuccess"),
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("maps STAGE_TEMPLATE_NAME_EXISTS to a friendly error", async () => {
    const user = userEvent.setup();
    mockCreate.mockRejectedValue(
      new ApiError("dup", "STAGE_TEMPLATE_NAME_EXISTS", 409),
    );
    renderWithClient(
      <CreateStageTemplateModal open={true} onOpenChange={vi.fn()} />,
    );

    await user.type(screen.getByLabelText(/stages\.stageName/), "Onsite");
    await user.type(screen.getByLabelText(/stages\.order/), "4");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("stages.nameExists"),
    );
  });

  it("falls back to the raw message on a generic create failure", async () => {
    const user = userEvent.setup();
    mockCreate.mockRejectedValue(new Error("server exploded"));
    renderWithClient(
      <CreateStageTemplateModal open={true} onOpenChange={vi.fn()} />,
    );

    await user.type(screen.getByLabelText(/stages\.stageName/), "Onsite");
    await user.type(screen.getByLabelText(/stages\.order/), "4");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("server exploded"),
    );
  });

  it("closes without submitting when cancel is clicked", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderWithClient(
      <CreateStageTemplateModal open={true} onOpenChange={onOpenChange} />,
    );
    await user.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(mockCreate).not.toHaveBeenCalled();
  });
});

describe("EditStageTemplateModal — behavior", () => {
  it("prefills the form from the template", () => {
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={template}
      />,
    );
    expect(screen.getByLabelText(/stages\.name/)).toHaveValue("Phone Screen");
    expect(screen.getByLabelText(/stages\.order/)).toHaveValue(2);
  });

  it("submits an update with the edited values", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockUpdate.mockResolvedValue(template);
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={onOpenChange}
        template={template}
      />,
    );

    const nameInput = screen.getByLabelText(/stages\.name/);
    await user.clear(nameInput);
    await user.type(nameInput, "Final Round");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("st-1", {
        name: "Final Round",
        order: 2,
      }),
    );
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("stages.updateSuccess"),
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("maps a duplicate-name error on update", async () => {
    const user = userEvent.setup();
    mockUpdate.mockRejectedValue(
      new ApiError("dup", "STAGE_TEMPLATE_NAME_EXISTS", 409),
    );
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={template}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("stages.nameExists"),
    );
  });

  it("surfaces a generic update error message", async () => {
    const user = userEvent.setup();
    mockUpdate.mockRejectedValue(new Error("nope"));
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={template}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() => expect(mockShowError).toHaveBeenCalledWith("nope"));
  });
});
