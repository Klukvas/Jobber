import { useState } from "react";
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
    expect(screen.getByLabelText(/stages\.order/)).toHaveValue(6);
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
    expect(mockCreate.mock.calls[0][0]).toEqual({ name: "Onsite", order: 3 });
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
    expect(screen.getByLabelText(/stages\.order/)).toHaveValue(3);
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

/**
 * A column's *meaning* is separate from its name and its position: analytics
 * counts kinds, so renaming "Technical Interview" or dragging it one slot left
 * must not change what it counts. A column the user invents starts as "Other",
 * which counts towards nothing until they classify it here.
 */
describe("stage kind", () => {
  it("starts a custom stage as Other and sends no kind for it", async () => {
    const user = userEvent.setup();
    mockCreate.mockResolvedValue({ ...template, id: "new" });
    renderWithClient(
      <CreateStageTemplateModal open={true} onOpenChange={vi.fn()} />,
    );

    expect(screen.getByLabelText("stages.kind")).toHaveValue("other");

    await user.type(screen.getByLabelText(/stages\.stageName/), "Follow-up");
    await user.type(screen.getByLabelText(/stages\.order/), "8");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    // Omitted, because omitted *is* "other" on the server — the column simply
    // carries no analytics meaning yet.
    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({
      name: "Follow-up",
      order: 7,
    });
  });

  it("offers every kind and sends the one the user picks", async () => {
    const user = userEvent.setup();
    mockCreate.mockResolvedValue({ ...template, id: "new" });
    renderWithClient(
      <CreateStageTemplateModal open={true} onOpenChange={vi.fn()} />,
    );

    const select = screen.getByLabelText("stages.kind");
    expect(
      Array.from(select.querySelectorAll("option")).map((o) => o.value),
    ).toEqual([
      "saved",
      "applied",
      "response",
      "interview",
      "offer",
      "rejected",
      "other",
    ]);

    await user.type(screen.getByLabelText(/stages\.stageName/), "Onsite loop");
    await user.type(screen.getByLabelText(/stages\.order/), "4");
    await user.selectOptions(select, "interview");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({
      name: "Onsite loop",
      order: 3,
      kind: "interview",
    });
  });

  it("keeps the kind when the stage is only renamed", async () => {
    const user = userEvent.setup();
    mockUpdate.mockResolvedValue(template);
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={{ ...template, kind: "interview" }}
      />,
    );

    expect(screen.getByLabelText("stages.kind")).toHaveValue("interview");

    const nameInput = screen.getByLabelText(/stages\.name/);
    await user.clear(nameInput);
    await user.type(nameInput, "Onsite loop");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    // No kind in the payload: a rename is not a reclassification, and the
    // server leaves an omitted kind alone.
    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("st-1", {
        name: "Onsite loop",
        order: 2,
      }),
    );
  });

  it("keeps the kind when the stage is only reordered", async () => {
    const user = userEvent.setup();
    mockUpdate.mockResolvedValue(template);
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={{ ...template, kind: "response" }}
      />,
    );

    const orderInput = screen.getByLabelText(/stages\.order/);
    await user.clear(orderInput);
    await user.type(orderInput, "5");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("st-1", {
        name: "Phone Screen",
        order: 4,
      }),
    );
  });

  it("sends the new kind when the user reclassifies the stage", async () => {
    const user = userEvent.setup();
    mockUpdate.mockResolvedValue(template);
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={{ ...template, kind: "other" }}
      />,
    );

    await user.selectOptions(screen.getByLabelText("stages.kind"), "response");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("st-1", {
        name: "Phone Screen",
        order: 2,
        kind: "response",
      }),
    );
  });

  // A column written before kinds existed arrives without one.
  it("reads a stage with no kind as Other and lets it be classified", async () => {
    const user = userEvent.setup();
    mockUpdate.mockResolvedValue(template);
    renderWithClient(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={template}
      />,
    );

    expect(screen.getByLabelText("stages.kind")).toHaveValue("other");

    await user.selectOptions(screen.getByLabelText("stages.kind"), "interview");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("st-1", {
        name: "Phone Screen",
        order: 2,
        kind: "interview",
      }),
    );
  });
});


function EditingSession() {
  const [open, setOpen] = useState(true);
  return <><button onClick={() => setOpen(true)}>Reopen</button><EditStageTemplateModal open={open} onOpenChange={setOpen} template={template} /></>;
}

describe("stage editing sessions", () => {
  it.each(["cancel", "close", "escape", "backdrop"])("discards a draft on %s and restores persisted values", async (method) => {
    const user = userEvent.setup();
    renderWithClient(<EditingSession />);
    await user.clear(screen.getByLabelText(/stages.name/));
    await user.type(screen.getByLabelText(/stages.name/), "Cancelled draft");
    if (method === "escape") await user.keyboard("{Escape}");
    else if (method === "backdrop") await user.click(screen.getByRole("dialog").parentElement!);
    else await user.click(screen.getByRole("button", { name: method === "cancel" ? "common.cancel" : "common.close" }));
    await user.click(screen.getByRole("button", { name: "Reopen" }));
    expect(screen.getByLabelText(/stages.name/)).toHaveValue(template.name);
    expect(mockUpdate).not.toHaveBeenCalled();
  });
  it("retains the entered draft after a failed save", async () => {
    const user = userEvent.setup();
    mockUpdate.mockRejectedValue(new Error("retry"));
    renderWithClient(<EditingSession />);
    await user.clear(screen.getByLabelText(/stages.name/));
    await user.type(screen.getByLabelText(/stages.name/), "Keep my draft");
    await user.click(screen.getByRole("button", { name: "common.save" }));
    await waitFor(() => expect(mockShowError).toHaveBeenCalled());
    expect(screen.getByLabelText(/stages.name/)).toHaveValue("Keep my draft");
  });
  it("an older save cannot close a reopened session", async () => {
    const user = userEvent.setup();
    let resolve!: (value: StageTemplateDTO) => void;
    mockUpdate.mockReturnValue(new Promise<StageTemplateDTO>((done) => { resolve = done; }));
    renderWithClient(<EditingSession />);
    await user.click(screen.getByRole("button", { name: "common.save" }));
    await user.click(screen.getByRole("button", { name: "common.cancel" }));
    await user.click(screen.getByRole("button", { name: "Reopen" }));
    resolve(template);
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(mockShowSuccess).not.toHaveBeenCalled();
  });
});
