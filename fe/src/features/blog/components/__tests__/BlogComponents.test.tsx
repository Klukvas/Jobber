import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { BlogHeader } from "../BlogHeader";
import { BlogPostCard } from "../BlogPostCard";
import { BlogArticle } from "../BlogArticle";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

vi.mock("marked", () => ({
  marked: {
    parse: (content: string) => `<p>${content}</p>`,
  },
}));

vi.mock("dompurify", () => ({
  default: {
    sanitize: (html: string) => html,
  },
}));

describe("BlogHeader", () => {
  it("renders title", () => {
    render(<BlogHeader />);
    expect(screen.getByText("blog.title")).toBeInTheDocument();
  });

  it("renders subtitle", () => {
    render(<BlogHeader />);
    expect(screen.getByText("blog.subtitle")).toBeInTheDocument();
  });
});

describe("BlogPostCard", () => {
  const mockPost = {
    slug: "test-post",
    title: "Test Post Title",
    description: "A test description",
    date: "2025-01-15T00:00:00Z",
    tags: ["React", "Testing"],
    lang: "en",
    content: "# Content",
  };

  it("renders post title", () => {
    render(<BlogPostCard post={mockPost} />);
    expect(screen.getByText("Test Post Title")).toBeInTheDocument();
  });

  it("renders post description", () => {
    render(<BlogPostCard post={mockPost} />);
    expect(screen.getByText("A test description")).toBeInTheDocument();
  });

  it("renders tags", () => {
    render(<BlogPostCard post={mockPost} />);
    expect(screen.getByText("React")).toBeInTheDocument();
    expect(screen.getByText("Testing")).toBeInTheDocument();
  });

  it("links to blog post", () => {
    render(<BlogPostCard post={mockPost} />);
    const link = screen.getByText("Test Post Title").closest("a");
    expect(link).toHaveAttribute("href", "/blog/test-post");
  });
});

describe("BlogArticle", () => {
  it("renders sanitized HTML content", () => {
    const { container } = render(<BlogArticle content="Hello world" />);
    expect(container.querySelector("article")).toBeTruthy();
    expect(container.textContent).toContain("Hello world");
  });
});
