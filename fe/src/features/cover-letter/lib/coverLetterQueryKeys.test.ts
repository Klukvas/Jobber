import { describe, it, expect } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import { coverLetterQueryKeys } from "./coverLetterQueryKeys";

/**
 * Asserted through a real QueryClient: which key an invalidation matches is
 * React Query's rule, not ours, and comparing the arrays would not prove it.
 *
 * A save invalidates the letters list on every rename. If the list key were a
 * prefix of the detail keys — `["cover-letter"]` and `["cover-letter", id]`
 * would be — that invalidation would sweep the letter the editor is showing as
 * well, and refetch it over the customer's cursor.
 */
describe("coverLetterQueryKeys", () => {
  const seeded = () => {
    const client = new QueryClient();
    client.setQueryData(coverLetterQueryKeys.list(), ["a letter"]);
    client.setQueryData(coverLetterQueryKeys.detail("cl-1"), "letter one");
    client.setQueryData(coverLetterQueryKeys.detail("cl-2"), "letter two");
    return client;
  };

  it("invalidates the list without touching a cached letter", async () => {
    const client = seeded();

    await client.invalidateQueries({ queryKey: coverLetterQueryKeys.list() });

    expect(
      client.getQueryState(coverLetterQueryKeys.list())?.isInvalidated,
    ).toBe(true);
    expect(
      client.getQueryState(coverLetterQueryKeys.detail("cl-1"))?.isInvalidated,
    ).toBe(false);
  });

  it("invalidates one letter without touching the others or the list", async () => {
    const client = seeded();

    await client.invalidateQueries({
      queryKey: coverLetterQueryKeys.detail("cl-1"),
      refetchType: "none",
    });

    expect(
      client.getQueryState(coverLetterQueryKeys.detail("cl-1"))?.isInvalidated,
    ).toBe(true);
    expect(
      client.getQueryState(coverLetterQueryKeys.detail("cl-2"))?.isInvalidated,
    ).toBe(false);
    expect(
      client.getQueryState(coverLetterQueryKeys.list())?.isInvalidated,
    ).toBe(false);
  });
});
