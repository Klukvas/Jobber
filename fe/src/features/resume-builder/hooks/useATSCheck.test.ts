import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useATSCheck } from "./useATSCheck";
import { apiClient } from "@/services/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

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
    return createElement(
      QueryClientProvider,
      { client: queryClient },
      children,
    );
  };
}

describe("useATSCheck", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls the correct API endpoint with resume ID", async () => {
    const mockResult = {
      score: 85,
      issues: [
        { severity: "warning" as const, description: "Missing keywords" },
      ],
      suggestions: ["Add more keywords"],
      keywords_found: ["React", "TypeScript"],
    };
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockResult);

    const { result } = renderHook(() => useATSCheck(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("resume-123");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(apiClient.post).toHaveBeenCalledWith(
      "resume-builder/resume-123/ats-check",
      { locale: "en" },
    );
    expect(result.current.data).toEqual(mockResult);
  });

  it("returns ATS check result with score, issues, suggestions, and keywords", async () => {
    const mockResult = {
      score: 92,
      issues: [],
      suggestions: [],
      keywords_found: ["JavaScript", "Node.js"],
    };
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockResult);

    const { result } = renderHook(() => useATSCheck(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("resume-456");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.score).toBe(92);
    expect(result.current.data?.issues).toEqual([]);
    expect(result.current.data?.suggestions).toEqual([]);
    expect(result.current.data?.keywords_found).toEqual([
      "JavaScript",
      "Node.js",
    ]);
  });

  it("handles critical severity issues", async () => {
    const mockResult = {
      score: 40,
      issues: [
        {
          severity: "critical" as const,
          description: "No contact information",
        },
        { severity: "info" as const, description: "Consider adding links" },
      ],
      suggestions: ["Add contact info"],
      keywords_found: [],
    };
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockResult);

    const { result } = renderHook(() => useATSCheck(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("resume-789");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const criticalIssues =
      result.current.data?.issues.filter((i) => i.severity === "critical") ??
      [];
    expect(criticalIssues).toHaveLength(1);
    expect(criticalIssues[0].description).toBe("No contact information");
  });

  it("sets isError on failure", async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error("Network error"));

    const { result } = renderHook(() => useATSCheck(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("resume-bad");
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeTruthy();
  });

  it("tracks loading state during mutation", async () => {
    let resolvePost: (val: unknown) => void = () => {};
    vi.mocked(apiClient.post).mockImplementationOnce(
      () =>
        new Promise((res) => {
          resolvePost = res;
        }),
    );

    const { result } = renderHook(() => useATSCheck(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isPending).toBe(false);

    act(() => {
      result.current.mutate("resume-loading");
    });

    await waitFor(() => expect(result.current.isPending).toBe(true));

    await act(async () => {
      resolvePost({
        score: 100,
        issues: [],
        suggestions: [],
        keywords_found: [],
      });
    });

    await waitFor(() => expect(result.current.isPending).toBe(false));
    expect(result.current.isSuccess).toBe(true);
  });

  it("is idle before any mutation", () => {
    const { result } = renderHook(() => useATSCheck(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isIdle).toBe(true);
    expect(result.current.data).toBeUndefined();
  });
});
