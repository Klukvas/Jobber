import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { createElement } from "react";

const { mockPost } = vi.hoisted(() => ({
  mockPost: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: { post: mockPost },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

import { useCoverLetterAI } from "../useCoverLetterAI";

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

describe("useCoverLetterAI", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls correct endpoint with request data", async () => {
    const requestData = {
      cover_letter_id: "cl-1",
      job_description: "Build great software",
    };

    mockPost.mockResolvedValueOnce({
      greeting: "Dear Hiring Manager,",
      paragraphs: ["Generated paragraph"],
      closing: "Best regards,",
    });

    const { result } = renderHook(() => useCoverLetterAI(), {
      wrapper: createWrapper(),
    });

    result.current.generate.mutate(requestData);

    await waitFor(() => {
      expect(result.current.generate.isSuccess).toBe(true);
    });

    expect(mockPost).toHaveBeenCalledWith(
      "cover-letters/ai/generate",
      requestData,
    );
  });

  it("returns generated cover letter data", async () => {
    const responseData = {
      greeting: "Dear Jane,",
      paragraphs: ["First paragraph", "Second paragraph"],
      closing: "Sincerely,",
    };

    mockPost.mockResolvedValueOnce(responseData);

    const { result } = renderHook(() => useCoverLetterAI(), {
      wrapper: createWrapper(),
    });

    result.current.generate.mutate({
      cover_letter_id: "cl-2",
    });

    await waitFor(() => {
      expect(result.current.generate.isSuccess).toBe(true);
    });

    expect(result.current.generate.data).toEqual(responseData);
  });

  it("sets isError on failure", async () => {
    mockPost.mockRejectedValueOnce(new Error("Network error"));

    const { result } = renderHook(() => useCoverLetterAI(), {
      wrapper: createWrapper(),
    });

    result.current.generate.mutate({
      cover_letter_id: "cl-3",
    });

    await waitFor(() => {
      expect(result.current.generate.isError).toBe(true);
    });

    expect(result.current.generate.error).toBeInstanceOf(Error);
  });
});
