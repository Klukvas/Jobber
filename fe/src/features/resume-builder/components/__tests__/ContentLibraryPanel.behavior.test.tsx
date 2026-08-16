import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ContentLibraryEntryDTO } from "@/shared/types/content-library";
import { ContentLibraryPanel } from "../ContentLibraryPanel";

const mockListSvc = vi.hoisted(() => vi.fn());
const mockCreate = vi.hoisted(() => vi.fn());
const mockUpdate = vi.hoisted(() => vi.fn());
const mockRemove = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("@/services/contentLibraryService", () => ({
  contentLibraryService: {
    list: mockListSvc,
    create: mockCreate,
    update: mockUpdate,
    remove: mockRemove,
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

function makeEntry(
  overrides?: Partial<ContentLibraryEntryDTO>,
): ContentLibraryEntryDTO {
  return {
    id: "e-1",
    title: "Leadership bullet",
    content: "Led a team of 5 engineers",
    category: "bullet",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ContentLibraryPanel />
    </QueryClientProvider>,
  );
}

const clipboardWrite = vi.fn().mockResolvedValue(undefined);

beforeEach(() => {
  vi.clearAllMocks();
  mockListSvc.mockResolvedValue([]);
  clipboardWrite.mockResolvedValue(undefined);
});

function installClipboard() {
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: clipboardWrite },
    configurable: true,
    writable: true,
  });
}

describe("ContentLibraryPanel", () => {
  it("shows the empty state when there are no snippets", async () => {
    renderPanel();
    expect(await screen.findByText("contentLibrary.empty")).toBeInTheDocument();
  });

  it("renders snippet cards from the service", async () => {
    mockListSvc.mockResolvedValue([
      makeEntry({ id: "a", title: "Alpha snippet" }),
      makeEntry({ id: "b", title: "Beta snippet", category: "summary" }),
    ]);
    renderPanel();
    expect(await screen.findByText("Alpha snippet")).toBeInTheDocument();
    expect(screen.getByText("Beta snippet")).toBeInTheDocument();
  });

  it("filters snippets by search text", async () => {
    const user = userEvent.setup();
    mockListSvc.mockResolvedValue([
      makeEntry({ id: "a", title: "Alpha snippet" }),
      makeEntry({ id: "b", title: "Beta snippet" }),
    ]);
    renderPanel();
    await screen.findByText("Alpha snippet");

    await user.type(
      screen.getByPlaceholderText("contentLibrary.search"),
      "Beta",
    );

    expect(screen.queryByText("Alpha snippet")).not.toBeInTheDocument();
    expect(screen.getByText("Beta snippet")).toBeInTheDocument();
  });

  it("filters snippets by category", async () => {
    const user = userEvent.setup();
    mockListSvc.mockResolvedValue([
      makeEntry({ id: "a", title: "Bullet one", category: "bullet" }),
      makeEntry({ id: "b", title: "Summary one", category: "summary" }),
    ]);
    renderPanel();
    await screen.findByText("Bullet one");

    // the first Select is the category filter
    const selects = screen.getAllByRole("combobox");
    await user.selectOptions(selects[0], "summary");

    expect(screen.queryByText("Bullet one")).not.toBeInTheDocument();
    expect(screen.getByText("Summary one")).toBeInTheDocument();
  });

  it("creates a snippet through the add dialog", async () => {
    const user = userEvent.setup();
    mockCreate.mockResolvedValue(makeEntry({ id: "new" }));
    renderPanel();
    await screen.findByText("contentLibrary.empty");

    await user.click(screen.getByText("contentLibrary.add"));

    const dialog = await screen.findByRole("dialog");
    const inputs = within(dialog).getAllByRole("textbox");
    // first textbox = title, second = content textarea
    await user.type(inputs[0], "New Title");
    await user.type(inputs[1], "New content body");
    await user.click(
      within(dialog).getByRole("button", { name: "contentLibrary.add" }),
    );

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({
      title: "New Title",
      content: "New content body",
      category: "bullet",
    });
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("contentLibrary.created"),
    );
  });

  it("keeps the save button disabled until title and content are filled", async () => {
    const user = userEvent.setup();
    renderPanel();
    await screen.findByText("contentLibrary.empty");
    await user.click(screen.getByText("contentLibrary.add"));

    const dialog = await screen.findByRole("dialog");
    const saveBtn = within(dialog).getByRole("button", {
      name: "contentLibrary.add",
    });
    expect(saveBtn).toBeDisabled();

    const inputs = within(dialog).getAllByRole("textbox");
    await user.type(inputs[0], "Only title");
    expect(saveBtn).toBeDisabled();
    await user.type(inputs[1], "now content");
    expect(saveBtn).toBeEnabled();
  });

  it("edits a snippet, prefilling and calling update", async () => {
    const user = userEvent.setup();
    mockListSvc.mockResolvedValue([
      makeEntry({ id: "edit-me", title: "Editable", content: "old body" }),
    ]);
    mockUpdate.mockResolvedValue(makeEntry());
    renderPanel();
    await screen.findByText("Editable");

    await user.click(screen.getByTitle("contentLibrary.edit"));

    const dialog = await screen.findByRole("dialog");
    const inputs = within(dialog).getAllByRole("textbox");
    expect(inputs[0]).toHaveValue("Editable");
    expect(inputs[1]).toHaveValue("old body");

    await user.clear(inputs[0]);
    await user.type(inputs[0], "Edited title");
    await user.click(
      within(dialog).getByRole("button", { name: "common.save" }),
    );

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("edit-me", {
        title: "Edited title",
        content: "old body",
        category: "bullet",
      }),
    );
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("contentLibrary.updated"),
    );
  });

  it("deletes a snippet after confirmation", async () => {
    const user = userEvent.setup();
    mockListSvc.mockResolvedValue([makeEntry({ id: "del-me" })]);
    mockRemove.mockResolvedValue(undefined);
    renderPanel();
    await screen.findByText("Leadership bullet");

    await user.click(screen.getByTitle("contentLibrary.delete"));

    const dialog = await screen.findByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: "contentLibrary.delete" }),
    );

    // RQ appends a mutation context arg to a bare mutationFn reference
    await waitFor(() => expect(mockRemove).toHaveBeenCalledTimes(1));
    expect(mockRemove.mock.calls[0][0]).toBe("del-me");
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("contentLibrary.deleted"),
    );
  });

  it("copies a snippet to the clipboard on insert", async () => {
    const user = userEvent.setup();
    installClipboard();
    mockListSvc.mockResolvedValue([
      makeEntry({ id: "ins", content: "Paste me" }),
    ]);
    renderPanel();
    await screen.findByText("Leadership bullet");

    await user.click(screen.getByTitle("contentLibrary.insert"));

    await waitFor(() =>
      expect(clipboardWrite).toHaveBeenCalledWith("Paste me"),
    );
    expect(mockShowSuccess).toHaveBeenCalledWith(
      "contentLibrary.insertSuccess",
    );
  });

  it("shows an error toast when the create request fails", async () => {
    const user = userEvent.setup();
    mockCreate.mockRejectedValue(new Error("boom"));
    renderPanel();
    await screen.findByText("contentLibrary.empty");
    await user.click(screen.getByText("contentLibrary.add"));

    const dialog = await screen.findByRole("dialog");
    const inputs = within(dialog).getAllByRole("textbox");
    await user.type(inputs[0], "T");
    await user.type(inputs[1], "C");
    await user.click(
      within(dialog).getByRole("button", { name: "contentLibrary.add" }),
    );

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("common.error"),
    );
  });
});
