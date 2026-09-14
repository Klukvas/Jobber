import { describe, it, expect, afterEach } from "vitest";
import { render } from "@testing-library/react";

import { BlogPostJsonLd } from "../BlogPostJsonLd";
import type { BlogPost } from "../../lib/blogLoader";

const post: BlogPost = {
  slug: "script-tags-and-seo",
  // Everything an author can type ends up inside a <script> element that the
  // prerenderer writes straight into the shipped HTML.
  title: 'Why </script><img src=x onerror="alert(1)"> breaks SEO',
  description: "Angle brackets < and > in prose",
  date: "2026-01-15",
  tags: ["<b>seo</b>"],
  lang: "en",
  content: "# Content",
};

afterEach(() => {
  document.getElementById("blog-post-jsonld")?.remove();
});

describe("BlogPostJsonLd", () => {
  it("emits structured data that cannot close its own script element", () => {
    render(<BlogPostJsonLd post={post} />);

    const script = document.getElementById("blog-post-jsonld");
    expect(script?.textContent).not.toContain("</script");
    expect(script?.textContent).not.toContain("<");
  });

  it("still describes the post exactly, escaping and all", () => {
    render(<BlogPostJsonLd post={post} />);

    const script = document.getElementById("blog-post-jsonld");
    const [blogPosting] = JSON.parse(script?.textContent ?? "[]");

    expect(blogPosting.headline).toBe(post.title);
    expect(blogPosting.description).toBe(post.description);
    expect(blogPosting.keywords).toBe("<b>seo</b>");
  });

  it("does not leave the element behind when the page unmounts", () => {
    const { unmount } = render(<BlogPostJsonLd post={post} />);
    unmount();

    expect(document.getElementById("blog-post-jsonld")).toBeNull();
  });
});
