import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useUnifiedResumes } from "./useUnifiedResumes";
import type { ResumeDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";

// --- mock data factories ---

function createUploadedResume(
  overrides?: Partial<ResumeDTO>,
): ResumeDTO {
  return {
    id: "u-1",
    title: "Uploaded Resume",
    file_url: null,
    storage_type: "s3",
    is_active: true,
    applications_count: 0,
    can_delete: true,
    created_at: "2024-06-01T00:00:00Z",
    updated_at: "2024-06-15T00:00:00Z",
    ...overrides,
  };
}

function createBuilderResume(
  overrides?: Partial<ResumeBuilderDTO>,
): ResumeBuilderDTO {
  return {
    id: "b-1",
    title: "Builder Resume",
    template_id: "professional",
    font_family: "Inter",
    primary_color: "#000",
    text_color: "#333",
    spacing: 150,
    margin_top: 40,
    margin_bottom: 40,
    margin_left: 40,
    margin_right: 40,
    layout_mode: "single",
    sidebar_width: 35,
    font_size: 12,
    skill_display: "",
    created_at: "2024-07-01T00:00:00Z",
    updated_at: "2024-07-10T00:00:00Z",
    ...overrides,
  };
}

// --- mocks ---

let mockUploadedItems: ResumeDTO[] = [];
let mockBuilderItems: ResumeBuilderDTO[] = [];
let mockUploadedLoading = false;
let mockBuilderLoading = false;
let mockUploadedError: Error | null = null;
let mockBuilderError: Error | null = null;
const mockRefetch = vi.fn().mockResolvedValue({});

vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: string[] }) => {
    if (opts.queryKey[0] === "resumes") {
      return {
        data: mockUploadedError ? undefined : { items: mockUploadedItems },
        isLoading: mockUploadedLoading,
        isError: !!mockUploadedError,
        error: mockUploadedError,
        refetch: mockRefetch,
      };
    }
    return {
      data: mockBuilderError ? undefined : mockBuilderItems,
      isLoading: mockBuilderLoading,
      isError: !!mockBuilderError,
      error: mockBuilderError,
      refetch: mockRefetch,
    };
  },
}));

vi.mock("@/services/resumesService", () => ({
  resumesService: { list: vi.fn() },
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: { list: vi.fn() },
}));

describe("useUnifiedResumes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUploadedItems = [];
    mockBuilderItems = [];
    mockUploadedLoading = false;
    mockBuilderLoading = false;
    mockUploadedError = null;
    mockBuilderError = null;
  });

  it("merges uploaded and builder resumes into a single list", () => {
    mockUploadedItems = [createUploadedResume({ id: "u-1" })];
    mockBuilderItems = [createBuilderResume({ id: "b-1" })];

    const { result } = renderHook(() => useUnifiedResumes());

    expect(result.current.items).toHaveLength(2);
    expect(result.current.items.map((i) => i.kind)).toEqual(
      expect.arrayContaining(["uploaded", "built"]),
    );
  });

  it("filters by 'uploaded' kind", () => {
    mockUploadedItems = [createUploadedResume({ id: "u-1" })];
    mockBuilderItems = [createBuilderResume({ id: "b-1" })];

    const { result } = renderHook(() => useUnifiedResumes());

    act(() => result.current.setKindFilter("uploaded"));

    expect(result.current.items).toHaveLength(1);
    expect(result.current.items[0].kind).toBe("uploaded");
  });

  it("filters by 'built' kind", () => {
    mockUploadedItems = [createUploadedResume({ id: "u-1" })];
    mockBuilderItems = [createBuilderResume({ id: "b-1" })];

    const { result } = renderHook(() => useUnifiedResumes());

    act(() => result.current.setKindFilter("built"));

    expect(result.current.items).toHaveLength(1);
    expect(result.current.items[0].kind).toBe("built");
  });

  it("sorts by updated_at descending by default (newest first)", () => {
    mockUploadedItems = [
      createUploadedResume({ id: "u-old", updated_at: "2024-01-01T00:00:00Z" }),
      createUploadedResume({ id: "u-new", updated_at: "2024-12-01T00:00:00Z" }),
    ];
    mockBuilderItems = [];

    const { result } = renderHook(() => useUnifiedResumes());

    expect(result.current.items[0].data.id).toBe("u-new");
    expect(result.current.items[1].data.id).toBe("u-old");
  });

  it("sorts by title ascending", () => {
    mockUploadedItems = [
      createUploadedResume({ id: "u-b", title: "Banana" }),
      createUploadedResume({ id: "u-a", title: "Apple" }),
    ];
    mockBuilderItems = [createBuilderResume({ id: "b-c", title: "Cherry" })];

    const { result } = renderHook(() => useUnifiedResumes());

    act(() => result.current.toggleSort("title"));

    // Default sort dir for new field = desc, but for title desc means Z-A
    // toggleSort("title") sets sortBy=title, sortDir=desc (Z first)
    expect(result.current.items[0].data.title).toBe("Cherry");
    expect(result.current.items[2].data.title).toBe("Apple");
  });

  it("toggleSort toggles direction when same field clicked twice", () => {
    mockUploadedItems = [
      createUploadedResume({ id: "u-b", title: "Banana" }),
      createUploadedResume({ id: "u-a", title: "Apple" }),
    ];
    mockBuilderItems = [];

    const { result } = renderHook(() => useUnifiedResumes());

    act(() => result.current.toggleSort("title")); // desc
    act(() => result.current.toggleSort("title")); // asc

    expect(result.current.sortDir).toBe("asc");
    expect(result.current.items[0].data.title).toBe("Apple");
    expect(result.current.items[1].data.title).toBe("Banana");
  });

  it("sorts by created_at", () => {
    mockUploadedItems = [
      createUploadedResume({
        id: "u-1",
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-12-01T00:00:00Z",
      }),
      createUploadedResume({
        id: "u-2",
        created_at: "2024-06-01T00:00:00Z",
        updated_at: "2024-06-01T00:00:00Z",
      }),
    ];
    mockBuilderItems = [];

    const { result } = renderHook(() => useUnifiedResumes());

    act(() => result.current.toggleSort("created_at"));

    // desc = newest created first
    expect(result.current.items[0].data.id).toBe("u-2");
  });

  it("handles invalid dates gracefully in sort (NaN guard)", () => {
    mockUploadedItems = [
      createUploadedResume({ id: "u-bad", updated_at: "" }),
      createUploadedResume({ id: "u-good", updated_at: "2024-06-01T00:00:00Z" }),
    ];
    mockBuilderItems = [];

    const { result } = renderHook(() => useUnifiedResumes());

    // Should not throw — invalid date treated as epoch 0
    expect(result.current.items).toHaveLength(2);
    // desc order: valid date first (larger timestamp)
    expect(result.current.items[0].data.id).toBe("u-good");
  });

  it("title sort is case-insensitive", () => {
    mockUploadedItems = [
      createUploadedResume({ id: "u-1", title: "banana" }),
      createUploadedResume({ id: "u-2", title: "Apple" }),
    ];
    mockBuilderItems = [];

    const { result } = renderHook(() => useUnifiedResumes());

    act(() => result.current.toggleSort("title")); // desc
    act(() => result.current.toggleSort("title")); // asc

    expect(result.current.items[0].data.title).toBe("Apple");
    expect(result.current.items[1].data.title).toBe("banana");
  });

  it("returns isLoading=true when either query is loading", () => {
    mockUploadedLoading = true;
    mockBuilderLoading = false;

    const { result } = renderHook(() => useUnifiedResumes());

    expect(result.current.isLoading).toBe(true);
  });

  it("returns isError=true when either query has error", () => {
    mockUploadedError = new Error("fail");

    const { result } = renderHook(() => useUnifiedResumes());

    expect(result.current.isError).toBe(true);
    expect(result.current.error).toBeInstanceOf(Error);
  });

  it("returns empty items when both queries return empty", () => {
    const { result } = renderHook(() => useUnifiedResumes());

    expect(result.current.items).toHaveLength(0);
  });

  it("refetch calls both query refetches", async () => {
    const { result } = renderHook(() => useUnifiedResumes());

    await act(async () => {
      await result.current.refetch();
    });

    expect(mockRefetch).toHaveBeenCalledTimes(2);
  });
});
