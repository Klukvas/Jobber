import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { JobContacts } from "./JobContacts";
import { contactsService, type Contact } from "@/services/contactsService";
vi.mock("@/services/contactsService", () => ({
  contactsService: { list: vi.fn(), save: vi.fn(), delete: vi.fn() },
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, args?: Record<string, string>) =>
      args?.name ? `${key} ${args.name}` : key,
  }),
}));
const contact: Contact = {
  id: "c1",
  job_id: "job1",
  name: "Ada",
  role: "Recruiter",
  email: "a@example.com",
  telegram: "https://t.me/ada_hr",
  linkedin: "https://www.linkedin.com/in/ada",
  note: "<script>alert(1)</script>",
  created_at: "now",
  updated_at: "now",
};
const setup = (jobId = "job1") => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const view = render(
    <QueryClientProvider client={client}>
      <JobContacts key={jobId} jobId={jobId} />
    </QueryClientProvider>,
  );
  return { client, ...view };
};
afterEach(cleanup);
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(contactsService.list).mockResolvedValue([]);
});

describe("Job contacts", () => {
  it("adds multiple contacts and exposes safe channels and plain notes", async () => {
    const user = userEvent.setup();
    vi.mocked(contactsService.save).mockResolvedValue(contact);
    setup();
    await user.click(
      await screen.findByRole("button", { name: "contacts.add" }),
    );
    const dialog = screen.getByRole("dialog");
    await user.type(
      within(dialog).getByLabelText("contacts.fields.name *"),
      "Ada",
    );
    await user.click(
      within(dialog).getByRole("button", { name: "common.save" }),
    );
    expect(await screen.findByText("Ada")).toBeVisible();
    expect(screen.getByRole("link", { name: "a@example.com" })).toHaveAttribute(
      "href",
      "mailto:a%40example.com",
    );
    expect(
      screen.getByRole("link", { name: "contacts.fields.telegram" }),
    ).toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.getByText(contact.note)).toBeVisible();
    expect(document.querySelector("script")).toBeNull();
    vi.mocked(contactsService.save).mockResolvedValue({
      ...contact,
      id: "c2",
      name: "Grace",
    });
    await user.click(screen.getByRole("button", { name: "contacts.add" }));
    await user.type(screen.getByLabelText("contacts.fields.name *"), "Grace");
    await user.click(screen.getByRole("button", { name: "common.save" }));
    expect(await screen.findByText("Grace")).toBeVisible();
    expect(screen.getByText("Ada")).toBeVisible();
  });
  it("keeps a failed save draft, validates fields and prevents duplicate submit", async () => {
    const user = userEvent.setup();
    let rejectSave: (reason: Error) => void = () => {};
    vi.mocked(contactsService.save).mockImplementation(
      () =>
        new Promise((_, reject) => {
          rejectSave = reject;
        }),
    );
    setup();
    await user.click(
      await screen.findByRole("button", { name: "contacts.add" }),
    );
    await user.click(screen.getByRole("button", { name: "common.save" }));
    expect(screen.getByText("contacts.required")).toBeVisible();
    expect(contactsService.save).not.toHaveBeenCalled();
    const name = screen.getByLabelText("contacts.fields.name *");
    await user.type(name, "Unsaved name");
    const save = screen.getByRole("button", { name: "common.save" });
    await user.dblClick(save);
    expect(contactsService.save).toHaveBeenCalledTimes(1);
    rejectSave(new Error("offline"));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "contacts.saveError",
    );
    expect(name).toHaveValue("Unsaved name");
    expect(screen.getByRole("dialog")).toBeVisible();
  });
  it("edits and confirms deletion; failure keeps the confirmation", async () => {
    const user = userEvent.setup();
    vi.mocked(contactsService.list).mockResolvedValue([contact]);
    vi.mocked(contactsService.save).mockResolvedValue({
      ...contact,
      name: "Ada updated",
    });
    vi.mocked(contactsService.delete)
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce();
    setup();
    await user.click(
      await screen.findByRole("button", {
        name: "contacts.edit",
        description: "Ada",
      }),
    );
    const name = screen.getByLabelText("contacts.fields.name *");
    await user.clear(name);
    await user.type(name, "Ada updated");
    await user.click(screen.getByRole("button", { name: "common.save" }));
    await user.click(
      await screen.findByRole("button", {
        name: "contacts.delete",
        description: "Ada updated",
      }),
    );
    expect(contactsService.delete).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "common.delete" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "contacts.deleteError",
    );
    await user.click(screen.getByRole("button", { name: "common.delete" }));
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(screen.getByText("contacts.empty")).toBeVisible();
  });
  it("focuses name immediately and keeps an immediate role edit after save fails", async () => {
    vi.mocked(contactsService.list).mockResolvedValue([contact]);
    vi.mocked(contactsService.save).mockRejectedValue(new Error("offline"));
    setup();
    const edit = await screen.findByRole("button", {
      name: "contacts.edit",
      description: "Ada",
    });
    fireEvent.click(edit);
    expect(screen.getByLabelText("contacts.fields.name *")).toHaveFocus();
    const role = screen.getByLabelText("contacts.fields.role");
    fireEvent.change(role, { target: { value: "Immediate role edit" } });
    fireEvent.click(screen.getByRole("button", { name: "common.save" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "contacts.saveError",
    );
    expect(role).toHaveValue("Immediate role edit");
    expect(contactsService.save).toHaveBeenCalledWith(
      "job1",
      "c1",
      expect.objectContaining({ role: "Immediate role edit" }),
    );
  });
  it("does not display a previous job's draft or delayed save on route switch", async () => {
    const user = userEvent.setup();
    let resolveSave: (value: Contact) => void = () => {};
    vi.mocked(contactsService.save).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveSave = resolve;
        }),
    );
    const { rerender, client } = setup();
    await user.click(
      await screen.findByRole("button", { name: "contacts.add" }),
    );
    await user.type(
      screen.getByLabelText("contacts.fields.name *"),
      "Old draft",
    );
    await user.click(screen.getByRole("button", { name: "common.save" }));
    rerender(
      <QueryClientProvider client={client}>
        <JobContacts key="job2" jobId="job2" />
      </QueryClientProvider>,
    );
    resolveSave(contact);
    expect(await screen.findByText("contacts.empty")).toBeVisible();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.queryByText("Ada")).toBeNull();
  });
  it("cancels a delayed list response before committing deletion", async () => {
    const user = userEvent.setup();
    vi.mocked(contactsService.list).mockResolvedValueOnce([contact]);
    vi.mocked(contactsService.delete).mockResolvedValue();
    const { client } = setup();
    await screen.findByText("Ada");
    let resolveList: (value: Contact[]) => void = () => {};
    vi.mocked(contactsService.list).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveList = resolve;
        }),
    );
    void client.refetchQueries({ queryKey: ["job-contacts", "job1"] });
    await user.click(
      screen.getByRole("button", {
        name: "contacts.delete",
        description: "Ada",
      }),
    );
    await user.click(screen.getByRole("button", { name: "common.delete" }));
    expect(await screen.findByText("contacts.empty")).toBeVisible();
    resolveList([contact]);
    await waitFor(() =>
      expect(client.getQueryData(["job-contacts", "job1"])).toEqual([]),
    );
  });
});
