import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { JobTags } from "../JobTags";
import type { TagDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockMutate = vi.hoisted(() => vi.fn());
const mockMutationState = vi.hoisted(() => ({ isPending: false }));
const mockQueries = vi.hoisted(() => ({
  jobTags: [] as TagDTO[],
  allTags: [] as TagDTO[],
}));
const mockInvalidate = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "job-tags") return { data: mockQueries.jobTags };
    return { data: mockQueries.allTags };
  },
  useMutation: (opts: {
    mutationFn: (v: unknown) => Promise<unknown>;
    onSuccess?: (r: unknown) => void;
    onError?: (e: Error) => void;
  }) => ({
    mutate: (v: unknown) => {
      mockMutate(v);
      return Promise.resolve(opts.mutationFn(v)).then(
        opts.onSuccess,
        opts.onError,
      );
    },
    isPending: mockMutationState.isPending,
  }),
  useQueryClient: () => ({ invalidateQueries: mockInvalidate }),
}));

const mockTagsService = vi.hoisted(() => ({
  listByJob: vi.fn(),
  list: vi.fn(),
  attach: vi.fn(),
  detach: vi.fn(),
  create: vi.fn(),
}));
vi.mock("@/services/tagsService", () => ({ tagsService: mockTagsService }));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

function tag(id: string, name: string): TagDTO {
  return {
    id,
    name,
    color: "#2563eb",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  } as TagDTO;
}

describe("JobTags", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
    mockQueries.jobTags = [];
    mockQueries.allTags = [];
  });

  it("shows an empty-state message when no tags are attached", () => {
    render(<JobTags jobId="job-1" />);
    expect(screen.getByText("jobs.noTags")).toBeInTheDocument();
  });

  it("renders attached tag chips and detaches on remove click", async () => {
    const user = userEvent.setup();
    mockQueries.jobTags = [tag("t1", "Remote")];
    mockTagsService.detach.mockResolvedValue(undefined);

    render(<JobTags jobId="job-1" />);
    expect(screen.getByText("Remote")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "common.delete" }));
    expect(mockTagsService.detach).toHaveBeenCalledWith("t1", "job", "job-1");
  });

  it("lists unattached tags and attaches one on click", async () => {
    const user = userEvent.setup();
    mockQueries.jobTags = [tag("t1", "Remote")];
    mockQueries.allTags = [tag("t1", "Remote"), tag("t2", "Urgent")];
    mockTagsService.attach.mockResolvedValue(undefined);

    render(<JobTags jobId="job-1" />);
    expect(screen.getByText("jobs.addExistingTag")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Urgent/ }));
    expect(mockTagsService.attach).toHaveBeenCalledWith("t2", {
      entity_type: "job",
      entity_id: "job-1",
    });
  });

  it("keeps the create button disabled until a name is entered", async () => {
    const user = userEvent.setup();
    render(<JobTags jobId="job-1" />);

    const create = screen.getByRole("button", { name: "common.create" });
    expect(create).toBeDisabled();

    await user.type(
      screen.getByPlaceholderText("jobs.newTagPlaceholder"),
      "Backend",
    );
    expect(create).toBeEnabled();
  });

  it("creates a new tag and attaches it, then clears the input", async () => {
    const user = userEvent.setup();
    mockTagsService.create.mockResolvedValue(tag("t9", "Backend"));
    mockTagsService.attach.mockResolvedValue(undefined);

    render(<JobTags jobId="job-1" />);
    const input = screen.getByPlaceholderText("jobs.newTagPlaceholder");
    await user.type(input, "Backend");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mockTagsService.create).toHaveBeenCalledWith({
        name: "Backend",
        color: "#2563eb",
      }),
    );
    await waitFor(() =>
      expect(mockTagsService.attach).toHaveBeenCalledWith("t9", {
        entity_type: "job",
        entity_id: "job-1",
      }),
    );
    expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
      "jobs.tagCreatedSuccess",
    );
  });
});
