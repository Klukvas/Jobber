import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { BlogPostCard } from "../BlogPostCard";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const posts = [
  {
    slug: "first-post",
    title: "First Post",
    description: "One",
    date: "2025-01-15T00:00:00Z",
    tags: ["React"],
    lang: "en",
    content: "# One",
  },
  {
    slug: "second-post",
    title: "Second Post",
    description: "Two",
    date: "2025-02-15T00:00:00Z",
    tags: [],
    lang: "en",
    content: "# Two",
  },
];

/**
 * Every card used to carry a second, empty `<a class="absolute inset-0">` over
 * the whole of it. It duplicated the destination of the title link right above
 * it and had nothing inside to name it, so `/blog` offered one anonymous link
 * per post — announced as "link", with no way to tell one from another.
 *
 * The real router is used rather than a stubbed `Link`, because the fix is in
 * the markup: a stub that renders `<a href>` and drops everything else would
 * pass whatever the component did.
 */
describe("BlogPostCard accessibility", () => {
  function renderCards() {
    return render(
      <MemoryRouter>
        {posts.map((post) => (
          <BlogPostCard key={post.slug} post={post} />
        ))}
      </MemoryRouter>,
    );
  }

  it("offers exactly one link per card", () => {
    renderCards();

    expect(screen.getAllByRole("link")).toHaveLength(posts.length);
  });

  it("names every link with the post it opens", () => {
    renderCards();

    expect(screen.getByRole("link", { name: "First Post" })).toHaveAttribute(
      "href",
      "/blog/first-post",
    );
    expect(screen.getByRole("link", { name: "Second Post" })).toHaveAttribute(
      "href",
      "/blog/second-post",
    );
  });

  it("leaves no anchor without an accessible name", () => {
    const { container } = renderCards();

    const unnamed = Array.from(container.querySelectorAll("a")).filter(
      (anchor) => anchor.textContent?.trim() === "",
    );
    expect(unnamed).toEqual([]);
  });

  it("hides no link from assistive technology", () => {
    const { container } = renderCards();

    expect(container.querySelectorAll('a[aria-hidden="true"]')).toHaveLength(0);
  });

  it("nests no link inside another", () => {
    const { container } = renderCards();

    expect(container.querySelectorAll("a a")).toHaveLength(0);
  });

  it("gives the keyboard one stop per card, in reading order", async () => {
    const user = userEvent.setup();
    const { container } = renderCards();

    await user.tab();
    expect(document.activeElement).toBe(
      screen.getByRole("link", { name: "First Post" }),
    );

    await user.tab();
    expect(document.activeElement).toBe(
      screen.getByRole("link", { name: "Second Post" }),
    );

    // Nothing else on the card answers a Tab — the anonymous overlays are gone.
    await user.tab();
    expect(container.contains(document.activeElement)).toBe(false);
  });

  it("keeps the whole card clickable", () => {
    renderCards();

    const link = screen.getByRole("link", { name: "First Post" });
    // The card-wide target is the link's own stretched pseudo-element, so the
    // area survives without a second anchor in the accessibility tree.
    expect(link.className).toContain("after:absolute");
    expect(link.className).toContain("after:inset-0");
    expect(link.closest("article")?.className).toContain("relative");
  });
});
