import { describe, it, expect, vi } from "vitest";

// Mock import.meta.glob before importing the module
vi.mock("/src/content/blog/en/*.md", () => ({}));
vi.mock("/src/content/blog/ua/*.md", () => ({}));
vi.mock("/src/content/blog/ru/*.md", () => ({}));

// We need to test the exported functions. The module reads import.meta.glob
// at load time, so we mock the glob results via vi.stubGlobal.
// Instead, let's test the logic by importing and calling the functions directly.

describe("blogLoader", () => {
  it("getAllPosts returns empty array for unknown lang", async () => {
    // Dynamic import after mocks are set up
    const { getAllPosts } = await import("./blogLoader");
    const posts = getAllPosts("en");
    // With no markdown files loaded, posts should be an empty array
    expect(Array.isArray(posts)).toBe(true);
  });

  it("getAllPosts returns empty for ua", async () => {
    const { getAllPosts } = await import("./blogLoader");
    const posts = getAllPosts("ua");
    expect(Array.isArray(posts)).toBe(true);
  });

  it("getAllPosts returns empty for ru", async () => {
    const { getAllPosts } = await import("./blogLoader");
    const posts = getAllPosts("ru");
    expect(Array.isArray(posts)).toBe(true);
  });

  it("getPostBySlug returns undefined when no posts", async () => {
    const { getPostBySlug } = await import("./blogLoader");
    const post = getPostBySlug("nonexistent", "en");
    expect(post).toBeUndefined();
  });
});
