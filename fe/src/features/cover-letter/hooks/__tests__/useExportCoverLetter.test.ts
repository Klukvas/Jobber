import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { createElement } from "react";

const { mockPostBlob } = vi.hoisted(() => ({
  mockPostBlob: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: {
    postBlob: mockPostBlob,
  },
}));

import { useExportCoverLetterPDF } from "../useExportCoverLetterPDF";
import { useExportCoverLetterDOCX } from "../useExportCoverLetterDOCX";

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

const mockBlob = new Blob(["test content"], { type: "application/pdf" });

describe("useExportCoverLetterPDF", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPostBlob.mockResolvedValue(mockBlob);
  });

  it("calls correct endpoint with cover letter ID", async () => {
    const { result } = renderHook(() => useExportCoverLetterPDF(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("cl-1");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(mockPostBlob).toHaveBeenCalledWith(
      "cover-letters/cl-1/export-pdf",
      undefined,
      60000,
    );
  });

  it("returns blob on success", async () => {
    const { result } = renderHook(() => useExportCoverLetterPDF(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("cl-1");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toBeInstanceOf(Blob);
  });
});

describe("useExportCoverLetterDOCX", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPostBlob.mockResolvedValue(mockBlob);
  });

  it("calls correct endpoint with cover letter ID", async () => {
    const { result } = renderHook(() => useExportCoverLetterDOCX(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("cl-2");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(mockPostBlob).toHaveBeenCalledWith(
      "cover-letters/cl-2/export-docx",
      undefined,
      60000,
    );
  });

  it("returns blob on success", async () => {
    const { result } = renderHook(() => useExportCoverLetterDOCX(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("cl-2");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toBeInstanceOf(Blob);
  });
});
