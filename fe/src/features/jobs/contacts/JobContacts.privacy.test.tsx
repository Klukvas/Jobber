import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as Sentry from "@sentry/react";
import { htmlTreeAsString } from "@sentry/core";
import { afterEach, describe, expect, it, vi } from "vitest";
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
  id: "contact-private",
  job_id: "job-private",
  name: "Sensitive Contact Name",
  role: "Sensitive Recruiting Role",
  email: "private.person@example.invalid",
  telegram: "https://t.me/private_person",
  linkedin: "https://www.linkedin.com/in/private-person",
  note: "Sensitive private note",
  created_at: "now",
  updated_at: "now",
};
const privateValues = [
  contact.name,
  contact.role,
  contact.email,
  encodeURIComponent(contact.email),
  contact.telegram,
  contact.linkedin,
  contact.note,
];
let replay: ReturnType<typeof Sentry.replayIntegration> | undefined;
afterEach(async () => {
  await replay?.stop();
  await Sentry.close();
  cleanup();
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});

// Exercise the installed SDK with an in-memory transport for all recordings
// and capture envelopes. No Sentry endpoint or external network is used.
describe("contact telemetry privacy", () => {
  it("blocks replay snapshots and keeps contact PII out of DOM breadcrumbs and capture envelopes", async () => {
    vi.mocked(contactsService.list).mockResolvedValue([contact]);
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={client}>
        <a href="mailto:unblocked-control@example.invalid">Replay control</a>
        <JobContacts jobId="job-private" />
      </QueryClientProvider>,
    );
    const edit = await screen.findByRole("button", {
      name: "contacts.edit",
      description: contact.name,
    });
    const remove = screen.getByRole("button", {
      name: "contacts.delete",
      description: contact.name,
    });
    expect(edit).toHaveAccessibleDescription(contact.name);
    expect(remove).toHaveAccessibleDescription(contact.name);
    expect(edit.getAttribute("aria-label")).toBe("contacts.edit");
    expect(edit.closest(".sentry-block")).not.toBeNull();

    // Enable the SDK browser path in jsdom; recorder and serializers stay real.
    vi.stubGlobal("__SENTRY_BROWSER_BUNDLE__", true);
    const recordings: string[] = [];
    const breadcrumbs: string[] = [];
    const envelopes: string[] = [];
    replay = Sentry.replayIntegration({
      maskAllText: true,
      blockAllMedia: true,
      useCompression: false,
      stickySession: false,
      minReplayDuration: 0,
      flushMinDelay: 100000,
      flushMaxDelay: 100000,
      beforeAddRecordingEvent: (event) => {
        recordings.push(JSON.stringify(event));
        return event;
      },
    });
    Sentry.init({
      dsn: "https://public@example.invalid/1",
      defaultIntegrations: false,
      integrations: [
        Sentry.breadcrumbsIntegration({
          console: false,
          fetch: false,
          xhr: false,
          history: false,
          dom: true,
        }),
        replay,
      ],
      replaysSessionSampleRate: 1,
      replaysOnErrorSampleRate: 0,
      beforeBreadcrumb: (breadcrumb) => {
        if (breadcrumb.category?.startsWith("ui."))
          breadcrumbs.push(JSON.stringify(breadcrumb));
        return breadcrumb;
      },
      transport: () => ({
        send: async (envelope) => {
          envelopes.push(JSON.stringify(envelope));
          return { statusCode: 200 };
        },
        flush: async () => true,
      }),
    });
    replay.start();
    await replay.flush();
    await waitFor(() =>
      expect(envelopes.join("")).toContain("unblocked-control@example.invalid"),
    );
    // A positive control proves href is retained by maskAllText alone, while
    // the actual contacts subtree is completely excluded by sentry-block.
    for (const value of privateValues)
      expect(envelopes.join("")).not.toContain(value);

    fireEvent.click(edit);
    const name = screen.getByLabelText("contacts.fields.name *");
    expect(name.closest(".sentry-block")).not.toBe(
      edit.closest(".sentry-block"),
    );
    expect(name.closest(".sentry-block")).not.toBeNull();
    for (const element of document.querySelectorAll(
      "button,input,textarea,a",
    )) {
      const serialized = htmlTreeAsString(element, { maxStringLength: 4096 });
      for (const value of privateValues)
        expect(serialized).not.toContain(value);
    }
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    fireEvent.click(remove);
    const confirmation = screen.getByText(
      `contacts.deleteConfirm ${contact.name}`,
    );
    expect(confirmation.closest(".sentry-block")).not.toBe(
      edit.closest(".sentry-block"),
    );
    expect(confirmation.closest(".sentry-block")).not.toBeNull();
    await waitFor(() => expect(breadcrumbs.length).toBeGreaterThan(0));
    await replay.flush();
    Sentry.captureMessage("contacts privacy regression");
    await Sentry.flush(1000);
    expect(envelopes.length).toBeGreaterThan(0);
    for (const recorded of [
      recordings.join(""),
      breadcrumbs.join(""),
      envelopes.join(""),
    ]) {
      for (const value of privateValues) expect(recorded).not.toContain(value);
    }
  });
});
