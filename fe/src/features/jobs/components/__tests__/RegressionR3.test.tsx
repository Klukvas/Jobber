import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CompanySelectWithQuickAdd } from "../CompanySelectWithQuickAdd";
import { JobListGrid } from "../JobListGrid";
import { JobPipelineCard } from "../JobPipelineCard";
import type { CompanyDTO, JobDTO, JobStageDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }) }));
const create = vi.hoisted(() => vi.fn());
vi.mock("@/services/companiesService", () => ({ companiesService: { create } }));
vi.mock("@/shared/lib/notifications", () => ({ showSuccessNotification: vi.fn(), showErrorNotification: vi.fn() }));
const job = { id: "j1", title: "Engineer", current_stage_id: "s1", current_stage_name: "Screening", created_at: "2026-01-01T00:00:00Z" } as JobDTO;
const stage = { id: "s1", status: "completed", stage_name: "Screening", completed_at: "2026-01-02T00:00:00Z" } as JobStageDTO;
const company = { id: "c1", name: "Company" } as CompanyDTO;
function client() {
  return new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
}
beforeEach(() => vi.clearAllMocks());

describe("R3 pipeline and list actions", () => {
  it("only offers completion for a loaded, unfinished current history stage", () => {
    const props = { job, templates: [], uploadedResumes: [], builderResumes: [], onMoveToStage: vi.fn(), isMoving: false, onCompleteStage: vi.fn(), isCompletingStage: false, onChangeResume: vi.fn(), isChangingResume: false };
    const view = render(<JobPipelineCard {...props} stages={[stage]} />);
    expect(screen.queryByRole("button", { name: "jobs.complete" })).not.toBeInTheDocument();
    view.rerender(<JobPipelineCard {...props} stages={[]} />);
    expect(screen.queryByRole("button", { name: "jobs.complete" })).not.toBeInTheDocument();
    view.rerender(<JobPipelineCard {...props} stages={[{ ...stage, status: "active", completed_at: undefined }]} />);
    fireEvent.click(screen.getByRole("button", { name: "jobs.complete" }));
    expect(props.onCompleteStage).toHaveBeenCalledWith("s1");
  });

  it("opens list actions with arrows, navigates, closes with Escape and restores focus", async () => {
    const user = userEvent.setup();
    const onToggleMenu = vi.fn();
    const props = { jobs: [job], isFetching: false, openMenuId: null as string | null, onToggleMenu, onAddComment: vi.fn(), onAddStage: vi.fn(), onDelete: vi.fn(), returnState: { from: "/app/jobs" } };
    const view = render(<MemoryRouter><JobListGrid {...props} /></MemoryRouter>);
    const trigger = screen.getByRole("button", { name: "jobs.actionsMenu" });
    trigger.focus();
    await user.keyboard("{ArrowDown}");
    expect(onToggleMenu).toHaveBeenCalledWith(job.id);
    view.rerender(<MemoryRouter><JobListGrid {...props} openMenuId={job.id} /></MemoryRouter>);
    const items = screen.getAllByRole("menuitem");
    expect(items[0]).toHaveFocus();
    await user.keyboard("{ArrowDown}");
    expect(items[1]).toHaveFocus();
    await user.keyboard("{End}");
    expect(items[2]).toHaveFocus();
    await user.keyboard("{Home}");
    expect(items[0]).toHaveFocus();
    await user.keyboard("{Escape}");
    expect(onToggleMenu).toHaveBeenLastCalledWith(null);
    expect(trigger).toHaveFocus();
  });
});

describe("R3 quick company creation", () => {
  function setup() {
    const onChange = vi.fn();
    render(<QueryClientProvider client={client()}><CompanySelectWithQuickAdd companies={[]} value="" onChange={onChange} /></QueryClientProvider>);
    fireEvent.click(screen.getByTitle("jobs.quickAddCompany"));
    return { onChange, input: screen.getByPlaceholderText("jobs.quickAddCompanyPlaceholder") };
  }
  it("rejects overlong names and exposes a field error", () => {
    const { input } = setup();
    fireEvent.change(input, { target: { value: "Ж".repeat(256) } });
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert")).toHaveTextContent("companies.nameTooLong");
    fireEvent.keyDown(input, { key: "Enter" });
    expect(create).not.toHaveBeenCalled();
  });
  it("allows 255 Unicode code points, trims and synchronously guards Enter plus click", async () => {
    let finish: (value: CompanyDTO) => void = () => {};
    create.mockReturnValue(new Promise<CompanyDTO>((resolve) => { finish = resolve; }));
    const { input, onChange } = setup();
    fireEvent.change(input, { target: { value: `  ${"🌍".repeat(255)}  ` } });
    act(() => {
      fireEvent.keyDown(input, { key: "Enter" });
      fireEvent.click(screen.getByRole("button", { name: "companies.create" }));
    });
    await waitFor(() => expect(create).toHaveBeenCalledTimes(1));
    expect(create).toHaveBeenCalledWith({ name: "🌍".repeat(255) });
    await act(async () => finish(company));
    await waitFor(() => expect(onChange).toHaveBeenCalledExactlyOnceWith("c1"));
  });
  it("preserves input on failure and permits retry", async () => {
    create.mockRejectedValueOnce(new Error("failed")).mockResolvedValueOnce(company);
    const { input, onChange } = setup();
    fireEvent.change(input, { target: { value: "  Company  " } });
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() => expect(screen.getByRole("button", { name: "companies.create" })).not.toBeDisabled());
    fireEvent.click(screen.getByRole("button", { name: "companies.create" }));
    await waitFor(() => expect(onChange).toHaveBeenCalledWith("c1"));
    expect(create).toHaveBeenCalledTimes(2);
  });
});
