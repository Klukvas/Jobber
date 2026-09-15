import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { BlogHeader } from "../BlogHeader";
import { BlogPostCard } from "../BlogPostCard";
import { BlogArticle } from "../BlogArticle";

/** The interface language, switched per test — the card dates from it. */
const uiLanguage = vi.hoisted(() => ({ value: "en" }));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: uiLanguage.value },
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

  afterEach(() => {
    uiLanguage.value = "en";
  });

  it("dates the card in English by default", () => {
    render(<BlogPostCard post={mockPost} />);
    expect(screen.getByText(/January 15th, 2025/)).toBeInTheDocument();
  });

  /**
   * The month name was translated but the *pattern* was not: `MMMM d, yyyy` is
   * the American order, so a Russian reader got "января 15, 2025" — a Russian
   * word in an English sentence, which is not how either language writes a
   * date. Nothing caught it, because asserting on the month name alone passes
   * whichever order it lands in.
   */
  it("dates the card the way Russian writes a date, day first", () => {
    uiLanguage.value = "ru";
    render(<BlogPostCard post={mockPost} />);

    expect(screen.getByText(/15 января 2025/)).toBeInTheDocument();
    expect(screen.queryByText(/января 15/)).not.toBeInTheDocument();
  });

  // date-fns writes the Ukrainian day as an ordinal — "15-е січня 2025 р." —
  // so the assertion is on the order, not on an exact spacing.
  it("dates the card the way Ukrainian writes a date, day first", () => {
    uiLanguage.value = "uk";
    render(<BlogPostCard post={mockPost} />);

    expect(screen.getByText(/15.{0,3} січня 2025/)).toBeInTheDocument();
    expect(screen.queryByText(/січня 15/)).not.toBeInTheDocument();
  });
});

describe("BlogArticle", () => {
  it("renders sanitized HTML content", () => {
    const { container } = render(<BlogArticle content="Hello world" />);
    expect(container.querySelector("article")).toBeTruthy();
    expect(container.textContent).toContain("Hello world");
  });
});
