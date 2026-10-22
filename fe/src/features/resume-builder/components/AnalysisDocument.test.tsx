import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AIAssistantPanel } from "./AIAssistantPanel";
import { ATSCheckerPanel } from "./ATSCheckerPanel";
import { createMockResume } from "./__tests__/testHelpers";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { apiClient } from "@/services/api";
import { flushResumeDocument } from "../lib/resumeDocumentSync";
import * as sectionPersistence from "../hooks/useSectionPersistence";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }),
}));
vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    canCreate: () => true,
    limits: { max_ai_requests: 1 },
  }),
}));
vi.mock("../lib/resumeDocumentSync", () => ({ flushResumeDocument: vi.fn() }));
vi.mock("@/services/api", async (original) => ({
  ...(await original<typeof import("@/services/api")>()),
  apiClient: { post: vi.fn() },
}));

function deferred<T>() {
  let resolve: (value: T) => void = () => {};
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
function mount(panel: "ai" | "ats") {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      {panel === "ai" ? <AIAssistantPanel /> : <ATSCheckerPanel />}
    </QueryClientProvider>,
  );
}
const score = {
  score: 87,
  issues: [],
  suggestions: ["Report from A"],
  keywords_found: [],
};

describe("analysis uses the saved document and keeps its result origin", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useResumeBuilderStore.getState().setResume(createMockResume({ id: "A" }));
    vi.mocked(flushResumeDocument).mockResolvedValue(undefined);
  });
  for (const panel of ["ai", "ats"] as const) {
    const action =
      panel === "ai"
        ? "resumeBuilder.ai.suggestSummary"
        : "resumeBuilder.ats.check";
    it(`${panel} waits for pending saves before calling the analysis API`, async () => {
      const save = deferred<void>();
      vi.mocked(flushResumeDocument).mockReturnValue(save.promise);
      vi.mocked(apiClient.post).mockResolvedValue(
        panel === "ai" ? { summary: "Generated summary" } : score,
      );
      mount(panel);
      await userEvent.click(screen.getByRole("button", { name: action }));
      expect(flushResumeDocument).toHaveBeenCalledWith("A");
      expect(apiClient.post).not.toHaveBeenCalled();
      await act(async () => save.resolve());
      await waitFor(() => expect(apiClient.post).toHaveBeenCalledTimes(1));
    });
    it(`${panel} aborts when saving fails`, async () => {
      vi.mocked(flushResumeDocument).mockRejectedValue(
        new Error("save failed"),
      );
      mount(panel);
      await userEvent.click(screen.getByRole("button", { name: action }));
      await waitFor(() =>
        expect(
          screen.getByText("resumeBuilder.analysis.saveError"),
        ).toBeInTheDocument(),
      );
      expect(apiClient.post).not.toHaveBeenCalled();
    });
    it(`${panel} does not start analysis after switching documents during save`, async () => {
      const save = deferred<void>();
      vi.mocked(flushResumeDocument).mockReturnValue(save.promise);
      mount(panel);
      await userEvent.click(screen.getByRole("button", { name: action }));
      act(() =>
        useResumeBuilderStore
          .getState()
          .setResume(createMockResume({ id: "B" })),
      );
      await act(async () => save.resolve());
      expect(apiClient.post).not.toHaveBeenCalled();
    });
    it(`${panel} ignores a late A response while cached B is active`, async () => {
      const reply = deferred<unknown>();
      vi.mocked(apiClient.post).mockReturnValue(reply.promise);
      mount(panel);
      await userEvent.click(screen.getByRole("button", { name: action }));
      await waitFor(() => expect(apiClient.post).toHaveBeenCalledTimes(1));
      act(() =>
        useResumeBuilderStore
          .getState()
          .setResume(
            createMockResume({ id: "B", summary: { content: "B original" } }),
          ),
      );
      await act(async () =>
        reply.resolve(panel === "ai" ? { summary: "Generated for A" } : score),
      );
      await waitFor(() =>
        expect(screen.getByRole("button", { name: action })).toBeEnabled(),
      );
      expect(
        screen.queryByText(
          panel === "ai" ? "Generated for A" : "Report from A",
        ),
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "resumeBuilder.ai.apply" }),
      ).not.toBeInTheDocument();
      expect(useResumeBuilderStore.getState().resume?.summary?.content).toBe(
        "B original",
      );
    });
    it(`${panel} hides results when the analyzed revision changes`, async () => {
      vi.mocked(apiClient.post).mockResolvedValue(
        panel === "ai" ? { summary: "Generated for A" } : score,
      );
      mount(panel);
      await userEvent.click(screen.getByRole("button", { name: action }));
      await screen.findByText(
        panel === "ai" ? "Generated for A" : "Report from A",
      );
      act(() =>
        useResumeBuilderStore
          .getState()
          .updateSummary({ content: "Newer edit" }),
      );
      expect(
        screen.queryByText(
          panel === "ai" ? "Generated for A" : "Report from A",
        ),
      ).not.toBeInTheDocument();
    });
  }
});

it("applies the current summary only to the analyzed revision", async () => {
  vi.clearAllMocks();
  useResumeBuilderStore.getState().setResume(createMockResume({ id: "A" }));
  vi.mocked(flushResumeDocument).mockResolvedValue(undefined);
  vi.mocked(apiClient.post).mockResolvedValue({
    summary: "Applicable suggestion",
  });
  mount("ai");
  await userEvent.click(
    screen.getByRole("button", { name: "resumeBuilder.ai.suggestSummary" }),
  );
  await screen.findByText("Applicable suggestion");
  await userEvent.click(
    screen.getByRole("button", { name: "resumeBuilder.ai.apply" }),
  );
  expect(useResumeBuilderStore.getState().resume?.summary?.content).toBe(
    "Applicable suggestion",
  );
});

it("never applies late bullets from A to an experience in B", async () => {
  vi.clearAllMocks();
  vi.mocked(flushResumeDocument).mockResolvedValue(undefined);
  const experience = {
    id: "exp-1",
    position: "Engineer",
    company: "Synthetic",
    description: "A original",
    location: "",
    start_date: "",
    end_date: "",
    is_current: false,
    sort_order: 0,
  };
  useResumeBuilderStore
    .getState()
    .setResume(createMockResume({ id: "A", experiences: [experience] }));
  const reply = deferred<unknown>();
  vi.mocked(apiClient.post).mockReturnValue(reply.promise);
  mount("ai");
  await userEvent.selectOptions(screen.getByRole("combobox"), "exp-1");
  await userEvent.click(
    screen.getByRole("button", { name: "resumeBuilder.ai.suggestBullets" }),
  );
  await waitFor(() => expect(apiClient.post).toHaveBeenCalledTimes(1));
  act(() =>
    useResumeBuilderStore.getState().setResume(
      createMockResume({
        id: "B",
        experiences: [{ ...experience, description: "B original" }],
      }),
    ),
  );
  await act(async () => reply.resolve({ bullets: ["A generated bullet"] }));
  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "resumeBuilder.ai.suggestBullets" }),
    ).toBeEnabled(),
  );
  expect(screen.queryByText("A generated bullet")).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "resumeBuilder.ai.apply" }),
  ).not.toBeInTheDocument();
  expect(
    useResumeBuilderStore.getState().resume?.experiences[0].description,
  ).toBe("B original");
});

it("keeps the selected experience when its CREATE receives a server ID during flush", async () => {
  vi.clearAllMocks();
  const experience = {
    id: "temp-exp",
    position: "Engineer",
    company: "Synthetic",
    description: "Original",
    location: "",
    start_date: "",
    end_date: "",
    is_current: false,
    sort_order: 0,
  };
  useResumeBuilderStore
    .getState()
    .setResume(createMockResume({ id: "A", experiences: [experience] }));
  const alias = vi
    .spyOn(sectionPersistence, "resolveResumeSectionIds")
    .mockImplementation((document) => ({
      ...document,
      experiences: document.experiences.map((entry) => ({
        ...entry,
        id: entry.id === "temp-exp" ? "server-exp" : entry.id,
      })),
    }));
  vi.mocked(flushResumeDocument).mockImplementation(async () => {
    useResumeBuilderStore
      .getState()
      .setResume(
        createMockResume({
          id: "A",
          experiences: [{ ...experience, id: "server-exp" }],
        }),
      );
  });
  vi.mocked(apiClient.post).mockResolvedValue({
    bullets: ["Current suggestion"],
  });
  mount("ai");
  await userEvent.selectOptions(screen.getByRole("combobox"), "temp-exp");
  await userEvent.click(
    screen.getByRole("button", { name: "resumeBuilder.ai.suggestBullets" }),
  );
  await screen.findByText("Current suggestion");
  expect(screen.getByRole("combobox")).toHaveValue("server-exp");
  await userEvent.click(
    screen.getByRole("button", { name: "resumeBuilder.ai.apply" }),
  );
  expect(useResumeBuilderStore.getState().resume?.experiences[0]).toMatchObject(
    { id: "server-exp", description: "• Current suggestion" },
  );
  alias.mockRestore();
});
