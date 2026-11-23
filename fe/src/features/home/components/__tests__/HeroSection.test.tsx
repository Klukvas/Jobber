import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { HeroSection } from "../HeroSection";
import { HERO_FRAME_MS } from "../HeroShowcase";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

describe("HeroSection", () => {
  const defaultProps = {
    isAuthenticated: false,
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it("renders headline translation keys", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.titleStart")).toBeInTheDocument();
    expect(screen.getByText("home.hero.titleAccent")).toBeInTheDocument();
  });

  it("renders subtitle", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.subtitle")).toBeInTheDocument();
  });

  it("renders badge", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.badge")).toBeInTheDocument();
  });

  it("shows register CTA when not authenticated", () => {
    const onRegister = vi.fn();
    render(
      <HeroSection
        isAuthenticated={false}
        onRegister={onRegister}
        onGoPlatform={vi.fn()}
      />,
    );
    // The primary button text is "home.hero.cta" + " →"
    const buttons = screen.getAllByRole("button");
    const primaryBtn = buttons.find(
      (btn) =>
        btn.textContent?.includes("home.hero.cta") &&
        !btn.textContent?.includes("Secondary"),
    );
    expect(primaryBtn).toBeTruthy();
    fireEvent.click(primaryBtn!);
    expect(onRegister).toHaveBeenCalledOnce();
  });

  it("shows secondary CTA when not authenticated", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.ctaSecondary")).toBeInTheDocument();
  });

  it("shows go-to-platform CTA when authenticated", () => {
    const onGoPlatform = vi.fn();
    render(
      <HeroSection
        isAuthenticated={true}
        onRegister={vi.fn()}
        onGoPlatform={onGoPlatform}
      />,
    );
    const btn = screen.getByText("home.hero.ctaGoPlatform");
    expect(btn).toBeInTheDocument();
    fireEvent.click(btn);
    expect(onGoPlatform).toHaveBeenCalledOnce();
  });

  it("does not show register CTA when authenticated", () => {
    render(
      <HeroSection
        isAuthenticated={true}
        onRegister={vi.fn()}
        onGoPlatform={vi.fn()}
      />,
    );
    expect(
      screen.queryByText("home.hero.ctaSecondary"),
    ).not.toBeInTheDocument();
  });

  it("renders the resume review preview instead of the board", () => {
    const { container } = render(<HeroSection {...defaultProps} />);
    const preview = activeFrame(container);
    expect(preview).toHaveTextContent("jobs.tailoring.review");
    expect(preview).toHaveTextContent("Staff Engineer — Vercel");
    expect(preview.querySelectorAll("article")).toHaveLength(2);
    expect(preview).toHaveTextContent("jobs.tailoring.before");
    expect(preview).toHaveTextContent("home.showcase.proposedEyebrow");
    expect(preview).toHaveTextContent("home.showcase.proposedTitle");
    expect(preview).toHaveTextContent("jobs.tailoring.reason");
    expect(preview).toHaveTextContent("home.showcase.summaryBefore");
    expect(preview).toHaveTextContent("home.showcase.summaryAfter");
    expect(preview).toHaveTextContent("home.showcase.summaryReason");
    expect(preview).not.toHaveTextContent("Stripe");
    expect(preview).not.toHaveTextContent("home.showcase.board");
  });

  it("shows the static review until a recording is configured", () => {
    const { container } = render(<HeroSection {...defaultProps} />);
    expect(
      container.querySelector("[data-hero-media-slot='resume-review']"),
    ).toBeTruthy();
    expect(container.querySelector("[data-hero-review='static']")).toBeTruthy();
    expect(container.querySelector("video")).toBeNull();
  });

  it("switches the right-hand frame from the tabs", () => {
    const { container } = render(<HeroSection {...defaultProps} />);
    expect(activeFrame(container)).toHaveAttribute("data-hero-frame", "edits");

    fireEvent.click(
      screen.getByRole("tab", { name: "home.showcase.frames.check" }),
    );
    expect(activeFrame(container)).toHaveAttribute("data-hero-frame", "check");
    expect(activeFrame(container)).toHaveTextContent(
      "home.showcase.check.note",
    );

    fireEvent.click(
      screen.getByRole("tab", { name: "home.showcase.frames.builder" }),
    );
    expect(activeFrame(container)).toHaveAttribute(
      "data-hero-frame",
      "builder",
    );
    expect(activeFrame(container)).toHaveTextContent(
      "home.showcase.builder.export",
    );

    fireEvent.click(
      screen.getByRole("tab", { name: "home.showcase.frames.stages" }),
    );
    expect(activeFrame(container)).toHaveAttribute("data-hero-frame", "stages");
    expect(activeFrame(container)).toHaveTextContent(
      "home.showcase.stages.offer",
    );
    expect(screen.queryByText("Stripe")).not.toBeInTheDocument();
  });

  it("advances the right-hand frame on a timer and pauses while hovered", () => {
    vi.useFakeTimers();
    try {
      const { container } = render(<HeroSection {...defaultProps} />);
      act(() => {
        vi.advanceTimersByTime(HERO_FRAME_MS);
      });
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "check",
      );

      const slot = container.querySelector(
        "[data-hero-media-slot='resume-review']",
      )!;
      fireEvent.mouseEnter(slot);
      act(() => {
        vi.advanceTimersByTime(HERO_FRAME_MS);
      });
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "check",
      );

      fireEvent.mouseLeave(slot);
      act(() => {
        vi.advanceTimersByTime(HERO_FRAME_MS);
      });
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "builder",
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it("keeps advancing after a pointer click once the pointer leaves", () => {
    vi.useFakeTimers();
    try {
      const { container } = render(<HeroSection {...defaultProps} />);
      const slot = container.querySelector(
        "[data-hero-media-slot='resume-review']",
      )!;
      const check = screen.getByRole("tab", {
        name: "home.showcase.frames.check",
      });
      fireEvent.mouseEnter(slot);
      fireEvent.pointerDown(check);
      fireEvent.focus(check);
      fireEvent.click(check);
      fireEvent.pointerUp(check);
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "check",
      );

      fireEvent.mouseLeave(slot);
      act(() => {
        vi.advanceTimersByTime(HERO_FRAME_MS);
      });
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "builder",
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it("pauses while a tab has keyboard focus and resumes when focus leaves", () => {
    vi.useFakeTimers();
    try {
      const { container } = render(<HeroSection {...defaultProps} />);
      const edits = screen.getByRole("tab", {
        name: "home.showcase.frames.edits",
      });
      fireEvent.focus(edits);
      act(() => {
        vi.advanceTimersByTime(HERO_FRAME_MS);
      });
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "edits",
      );

      fireEvent.blur(edits);
      act(() => {
        vi.advanceTimersByTime(HERO_FRAME_MS);
      });
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "check",
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it("does not auto-advance when reduced motion is requested", () => {
    const original = window.matchMedia;
    window.matchMedia = ((query: string) => ({
      matches: query.includes("prefers-reduced-motion"),
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    })) as typeof window.matchMedia;
    vi.useFakeTimers();
    try {
      const { container } = render(<HeroSection {...defaultProps} />);
      act(() => {
        vi.advanceTimersByTime(HERO_FRAME_MS * HERO_FRAMES_GUARD);
      });
      expect(activeFrame(container)).toHaveAttribute(
        "data-hero-frame",
        "edits",
      );
    } finally {
      vi.useRealTimers();
      window.matchMedia = original;
    }
  });
});

const HERO_FRAMES_GUARD = 4;

function activeFrame(container: HTMLElement) {
  const panel = container.querySelector("[data-active='true']");
  if (!panel) throw new Error("expected an active hero frame");
  return panel;
}
