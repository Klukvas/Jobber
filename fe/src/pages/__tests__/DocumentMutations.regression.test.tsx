import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ResumesPage from "../Resumes";
import CoverLettersPage from "../CoverLetters";
import { createMockResume } from "@/features/resume-builder/components/__tests__/testHelpers";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { coverLetterService } from "@/services/coverLetterService";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }), getI18n: () => ({ language: "en" }) }));
vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/notifications", () => ({ showSuccessNotification: vi.fn(), showErrorNotification: vi.fn() }));
vi.mock("@/shared/hooks/useSubscription", () => ({ useSubscription: () => ({ canCreate: () => true }) }));
vi.mock("@/features/resume-builder/components/ResumeThumbnail", () => ({ ResumeThumbnail: () => null }));
vi.mock("@/features/cover-letter/components/CoverLetterTemplateThumbnail", () => ({ CoverLetterTemplateThumbnail: () => null }));
vi.mock("@/features/resumes/hooks/useUnifiedResumes", () => ({ useUnifiedResumes: () => ({
  items: [{ kind: "built", data: createMockResume({ id: "builder", title: "Builder", updated_at: "2026-01-01T00:00:00Z" }) }],
  isLoading: false, isError: false, kindFilter: "all", setKindFilter: vi.fn(), sortBy: "created_at", sortOrder: "desc", toggleSort: vi.fn(), refetch: vi.fn(),
}) }));
vi.mock("@/services/resumeBuilderService", () => ({ resumeBuilderService: { duplicate: vi.fn() } }));
vi.mock("@/services/coverLetterService", () => ({ coverLetterService: { list: vi.fn(), delete: vi.fn() } }));

function deferred<T>() {
  let resolve: (value: T) => void = () => {};
  let reject: (reason: Error) => void = () => {};
  const promise = new Promise<T>((done, fail) => { resolve = done; reject = fail; });
  return { promise, resolve, reject };
}
function mount(element: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<MemoryRouter><QueryClientProvider client={client}>{element}</QueryClientProvider></MemoryRouter>);
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(coverLetterService.list).mockResolvedValue([{ id: "letter", title: "Letter", company_name: "", template: "professional", primary_color: "#2563eb", updated_at: "2026-01-01T00:00:00Z" }] as Awaited<ReturnType<typeof coverLetterService.list>>);
});

describe("document single-flight mutations", () => {
  it("sends one duplicate request for two synchronous clicks and unlocks after an error", async () => {
    const request = deferred<Awaited<ReturnType<typeof resumeBuilderService.duplicate>>>();
    vi.mocked(resumeBuilderService.duplicate).mockReturnValueOnce(request.promise).mockResolvedValue(createMockResume());
    mount(<ResumesPage />);
    const duplicate = screen.getByRole("button", { name: "resumeBuilder.duplicate" });
    fireEvent.click(duplicate);
    fireEvent.click(duplicate);
    await waitFor(() => expect(resumeBuilderService.duplicate).toHaveBeenCalledTimes(1));
    expect(duplicate).toBeDisabled();
    request.reject(new Error("offline"));
    await waitFor(() => expect(duplicate).toBeEnabled());
    await userEvent.click(duplicate);
    await waitFor(() => expect(resumeBuilderService.duplicate).toHaveBeenCalledTimes(2));
  });

  it("sends one delete request for two confirmations, then permits retry", async () => {
    const request = deferred<void>();
    vi.mocked(coverLetterService.delete).mockReturnValueOnce(request.promise).mockResolvedValue(undefined);
    mount(<CoverLettersPage />);
    await userEvent.click(await screen.findByRole("button", { name: "common.delete" }));
    const confirm = within(screen.getByRole("dialog")).getByRole("button", { name: "common.delete" });
    fireEvent.click(confirm);
    fireEvent.click(confirm);
    await waitFor(() => expect(coverLetterService.delete).toHaveBeenCalledTimes(1));
    request.reject(new Error("offline"));
    await waitFor(() => expect(confirm).toBeEnabled());
    await userEvent.click(confirm);
    await waitFor(() => expect(coverLetterService.delete).toHaveBeenCalledTimes(2));
  });
});
