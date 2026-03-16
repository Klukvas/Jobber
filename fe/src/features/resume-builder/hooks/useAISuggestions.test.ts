import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useAISuggestions } from "./useAISuggestions";
import { apiClient } from "@/services/api";

vi.mock("@/services/api", () => ({
  apiClient: {
    post: vi.fn(),
  },
  ApiError: class extends Error {
    code: string;
    status: number;
    constructor(message: string, code: string, status: number) {
      super(message);
      this.code = code;
      this.status = status;
    }
  },
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useAISuggestions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("suggestBullets", () => {
    it("calls the correct API endpoint with request data", async () => {
      const mockResponse = { bullets: ["bullet 1", "bullet 2"] };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const { result } = renderHook(() => useAISuggestions(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.suggestBullets.mutate({
          job_title: "Engineer",
          company: "Acme",
          current_description: "Did things",
        });
      });

      await waitFor(() =>
        expect(result.current.suggestBullets.isSuccess).toBe(true),
      );

      expect(apiClient.post).toHaveBeenCalledWith(
        "resume-builder/ai/suggest-bullets",
        {
          job_title: "Engineer",
          company: "Acme",
          current_description: "Did things",
        },
      );
      expect(result.current.suggestBullets.data).toEqual(mockResponse);
    });

    it("sets isError on failure", async () => {
      vi.mocked(apiClient.post).mockRejectedValueOnce(new Error("API error"));

      const { result } = renderHook(() => useAISuggestions(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.suggestBullets.mutate({
          job_title: "Engineer",
          company: "Acme",
          current_description: "",
        });
      });

      await waitFor(() =>
        expect(result.current.suggestBullets.isError).toBe(true),
      );

      expect(result.current.suggestBullets.error).toBeTruthy();
    });

    it("tracks loading state", async () => {
      let resolvePost: (val: unknown) => void = () => {};
      vi.mocked(apiClient.post).mockImplementationOnce(
        () => new Promise((res) => { resolvePost = res; }),
      );

      const { result } = renderHook(() => useAISuggestions(), {
        wrapper: createWrapper(),
      });

      expect(result.current.suggestBullets.isPending).toBe(false);

      act(() => {
        result.current.suggestBullets.mutate({
          job_title: "Engineer",
          company: "Acme",
          current_description: "",
        });
      });

      await waitFor(() =>
        expect(result.current.suggestBullets.isPending).toBe(true),
      );

      await act(async () => {
        resolvePost({ bullets: [] });
      });

      await waitFor(() =>
        expect(result.current.suggestBullets.isPending).toBe(false),
      );
    });
  });

  describe("suggestSummary", () => {
    it("calls the correct API endpoint", async () => {
      const mockResponse = { summary: "A great professional..." };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const { result } = renderHook(() => useAISuggestions(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.suggestSummary.mutate({ resume_id: "r-1" });
      });

      await waitFor(() =>
        expect(result.current.suggestSummary.isSuccess).toBe(true),
      );

      expect(apiClient.post).toHaveBeenCalledWith(
        "resume-builder/ai/suggest-summary",
        { resume_id: "r-1" },
      );
      expect(result.current.suggestSummary.data).toEqual(mockResponse);
    });

    it("handles error state", async () => {
      vi.mocked(apiClient.post).mockRejectedValueOnce(new Error("fail"));

      const { result } = renderHook(() => useAISuggestions(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.suggestSummary.mutate({ resume_id: "r-1" });
      });

      await waitFor(() =>
        expect(result.current.suggestSummary.isError).toBe(true),
      );
    });
  });

  describe("improveText", () => {
    it("calls the correct API endpoint", async () => {
      const mockResponse = { improved: "Better text here" };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const { result } = renderHook(() => useAISuggestions(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.improveText.mutate({
          text: "Original text",
          instruction: "Make it professional",
        });
      });

      await waitFor(() =>
        expect(result.current.improveText.isSuccess).toBe(true),
      );

      expect(apiClient.post).toHaveBeenCalledWith(
        "resume-builder/ai/improve-text",
        { text: "Original text", instruction: "Make it professional" },
      );
      expect(result.current.improveText.data).toEqual(mockResponse);
    });

    it("handles error state", async () => {
      vi.mocked(apiClient.post).mockRejectedValueOnce(new Error("fail"));

      const { result } = renderHook(() => useAISuggestions(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.improveText.mutate({
          text: "text",
          instruction: "improve",
        });
      });

      await waitFor(() =>
        expect(result.current.improveText.isError).toBe(true),
      );
    });
  });

  it("returns three mutation objects", () => {
    const { result } = renderHook(() => useAISuggestions(), {
      wrapper: createWrapper(),
    });

    expect(result.current.suggestBullets).toBeDefined();
    expect(result.current.suggestSummary).toBeDefined();
    expect(result.current.improveText).toBeDefined();
  });
});
