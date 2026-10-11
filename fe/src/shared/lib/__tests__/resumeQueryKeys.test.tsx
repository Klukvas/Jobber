import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from "@tanstack/react-query";
import { resumeQueryKeys } from "../resumeQueryKeys";

/**
 * The factory exists to make one invalidation reach every screen that shows
 * builder resumes, without also sweeping the editors' cached documents.
 *
 * The list used to be spelled `["resume-builders"]` on the resumes page and
 * `["resume-builder"]` in the job screens, so a resume created on one page
 * never appeared in the other's picker. The singular spelling is also a prefix
 * of every `["resume-builder", id]` document, so invalidating *that* list
 * dropped every open editor's cache too.
 *
 * These are asserted through a real QueryClient — which keys match which is
 * React Query's rule, not ours, and a shape assertion would not prove it.
 */

function listFetcher() {
  let calls = 0;
  return {
    get calls() {
      return calls;
    },
    fn: vi.fn(async () => {
      calls += 1;
      return `list#${calls}`;
    }),
  };
}

/** The resumes page and the job-screen picker: two screens, one query key. */
function Screens({
  listFn,
  detailFn,
}: {
  listFn: () => Promise<string>;
  detailFn: () => Promise<string>;
}) {
  const resumesPage = useQuery({
    queryKey: resumeQueryKeys.builderList(),
    queryFn: listFn,
  });
  const jobPicker = useQuery({
    queryKey: resumeQueryKeys.builderList(),
    queryFn: listFn,
  });
  const editor = useQuery({
    queryKey: resumeQueryKeys.builderDetail("rb-1"),
    queryFn: detailFn,
  });

  return (
    <div>
      <span data-testid="resumes-page">{resumesPage.data}</span>
      <span data-testid="job-picker">{jobPicker.data}</span>
      <span data-testid="editor">{editor.data}</span>
    </div>
  );
}

function renderScreens() {
  const list = listFetcher();
  const detail = listFetcher();
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <Screens listFn={list.fn} detailFn={detail.fn} />
    </QueryClientProvider>,
  );

  return { queryClient, list, detail };
}

describe("builder-resume query keys", () => {
  it("serves every screen from one cache entry", async () => {
    const { list } = renderScreens();

    await waitFor(() =>
      expect(screen.getByTestId("resumes-page")).toHaveTextContent("list#1"),
    );
    expect(screen.getByTestId("job-picker")).toHaveTextContent("list#1");
    // Two screens, one request: the key is shared, not merely similar.
    expect(list.calls).toBe(1);
  });

  it("refreshes every screen when the list is invalidated", async () => {
    const { queryClient, list, detail } = renderScreens();

    await waitFor(() => expect(list.calls).toBe(1));
    await waitFor(() =>
      expect(screen.getByTestId("editor")).toHaveTextContent("list#1"),
    );

    // What create / duplicate / delete / rename do.
    await queryClient.invalidateQueries({
      queryKey: resumeQueryKeys.builderList(),
    });

    await waitFor(() =>
      expect(screen.getByTestId("job-picker")).toHaveTextContent("list#2"),
    );
    expect(screen.getByTestId("resumes-page")).toHaveTextContent("list#2");
    // …and the open document was not thrown away along with it.
    expect(detail.calls).toBe(1);
  });

  it("does not refetch the list when one document is invalidated", async () => {
    const { queryClient, list, detail } = renderScreens();

    await waitFor(() => expect(list.calls).toBe(1));
    await waitFor(() => expect(detail.calls).toBe(1));

    await queryClient.invalidateQueries({
      queryKey: resumeQueryKeys.builderDetail("rb-1"),
    });

    await waitFor(() => expect(detail.calls).toBe(2));
    expect(list.calls).toBe(1);
  });

  it("keeps one document's invalidation off every other document", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(resumeQueryKeys.builderDetail("rb-1"), "one");
    queryClient.setQueryData(resumeQueryKeys.builderDetail("rb-2"), "two");

    await queryClient.invalidateQueries({
      queryKey: resumeQueryKeys.builderDetail("rb-1"),
    });

    expect(
      queryClient.getQueryState(resumeQueryKeys.builderDetail("rb-1"))
        ?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(resumeQueryKeys.builderDetail("rb-2"))
        ?.isInvalidated,
    ).toBe(false);
  });
});
