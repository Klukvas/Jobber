import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ContentLibraryPanel } from "./ContentLibraryPanel";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: [],
    isLoading: false,
  }),
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/services/contentLibraryService", () => ({
  contentLibraryService: {
    list: vi.fn().mockResolvedValue([]),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/shared/types/content-library", () => ({
  CONTENT_LIBRARY_CATEGORIES: ["bullet", "summary", "skill"],
}));

describe("ContentLibraryPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the content library title", () => {
    render(<ContentLibraryPanel />);
    expect(screen.getByText("contentLibrary.title")).toBeInTheDocument();
  });

  it("renders the add button", () => {
    render(<ContentLibraryPanel />);
    expect(screen.getByText("contentLibrary.add")).toBeInTheDocument();
  });

  it("renders the search input", () => {
    render(<ContentLibraryPanel />);
    expect(
      screen.getByPlaceholderText("contentLibrary.search"),
    ).toBeInTheDocument();
  });

  it("renders the category filter dropdown", () => {
    render(<ContentLibraryPanel />);
    expect(
      screen.getByText("contentLibrary.category"),
    ).toBeInTheDocument();
  });

  it("renders category options in the filter dropdown", () => {
    render(<ContentLibraryPanel />);
    expect(
      screen.getByText("contentLibrary.categories.bullet"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("contentLibrary.categories.summary"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("contentLibrary.categories.skill"),
    ).toBeInTheDocument();
  });
});
