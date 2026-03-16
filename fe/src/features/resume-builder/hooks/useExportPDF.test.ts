import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useExportPDF } from "./useExportPDF";
import { apiClient } from "@/services/api";

vi.mock("@/services/api", () => ({
  apiClient: {
    postBlob: vi.fn(),
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

describe("useExportPDF", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls postBlob with the correct endpoint and timeout", async () => {
    const mockBlob = new Blob(["pdf-data"], { type: "application/pdf" });
    vi.mocked(apiClient.postBlob).mockResolvedValueOnce(mockBlob);

    const { result } = renderHook(() => useExportPDF(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("resume-1");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(apiClient.postBlob).toHaveBeenCalledWith(
      "resume-builder/resume-1/export-pdf",
      undefined,
      60000,
    );
  });

  it("returns a Blob on success", async () => {
    const mockBlob = new Blob(["pdf-content"], { type: "application/pdf" });
    vi.mocked(apiClient.postBlob).mockResolvedValueOnce(mockBlob);

    const { result } = renderHook(() => useExportPDF(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("resume-2");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toBeInstanceOf(Blob);
  });

  it("sets isError on failure", async () => {
    vi.mocked(apiClient.postBlob).mockRejectedValueOnce(
      new Error("Export failed"),
    );

    const { result } = renderHook(() => useExportPDF(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("resume-fail");
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeTruthy();
  });

  it("tracks loading state", async () => {
    let resolveBlob: (val: Blob) => void = () => {};
    vi.mocked(apiClient.postBlob).mockImplementationOnce(
      () => new Promise<Blob>((res) => { resolveBlob = res; }),
    );

    const { result } = renderHook(() => useExportPDF(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isPending).toBe(false);

    act(() => {
      result.current.mutate("resume-loading");
    });

    await waitFor(() => expect(result.current.isPending).toBe(true));

    await act(async () => {
      resolveBlob(new Blob(["data"]));
    });

    await waitFor(() => expect(result.current.isPending).toBe(false));
  });

  it("is idle before mutation", () => {
    const { result } = renderHook(() => useExportPDF(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isIdle).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it("uses the resume ID in the URL path", async () => {
    vi.mocked(apiClient.postBlob).mockResolvedValueOnce(new Blob([]));

    const { result } = renderHook(() => useExportPDF(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate("abc-def-123");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(apiClient.postBlob).toHaveBeenCalledWith(
      "resume-builder/abc-def-123/export-pdf",
      undefined,
      60000,
    );
  });
});
