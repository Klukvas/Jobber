import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { SectionHeader } from "./SectionHeader";

describe("SectionHeader", () => {
  const defaultProps = {
    title: "Experience",
    color: "#e11d48",
    editable: false,
  };

  // ---------------------------------------------------------------------------
  // Variant-specific styling
  // ---------------------------------------------------------------------------

  describe("bold variant", () => {
    it("renders with colored background and white text by default", () => {
      render(<SectionHeader {...defaultProps} variant="bold" />);
      const heading = screen.getByText("Experience");
      expect(heading.className).toContain("rounded");
      expect(heading.className).toContain("uppercase");
      expect(heading.style.backgroundColor).toBe("rgb(225, 29, 72)");
      expect(heading.style.color).toBe("rgb(255, 255, 255)");
    });
  });

  describe("vivid variant", () => {
    it("renders as rounded pill with colored background and white text by default", () => {
      render(<SectionHeader {...defaultProps} variant="vivid" />);
      const heading = screen.getByText("Experience");
      expect(heading.className).toContain("rounded-full");
      expect(heading.className).toContain("text-xs");
      expect(heading.style.backgroundColor).toBe("rgb(225, 29, 72)");
      expect(heading.style.color).toBe("rgb(255, 255, 255)");
    });
  });

  describe("accent variant", () => {
    it("renders with left border and tinted background", () => {
      render(<SectionHeader {...defaultProps} variant="accent" />);
      const heading = screen.getByText("Experience");
      expect(heading.className).toContain("border-l-4");
      expect(heading.className).toContain("bg-gray-50");
      expect(heading.style.borderColor).toBe("rgb(225, 29, 72)");
      expect(heading.style.color).toBe("rgb(225, 29, 72)");
      expect(heading.style.backgroundColor).toBe("");
    });
  });

  describe("timeline variant", () => {
    it("renders with colored underline and text", () => {
      render(<SectionHeader {...defaultProps} variant="timeline" />);
      const heading = screen.getByText("Experience");
      expect(heading.className).toContain("uppercase");
      expect(heading.className).toContain("tracking-wider");
      expect(heading.style.borderColor).toBe("rgb(225, 29, 72)");
      expect(heading.style.color).toBe("rgb(225, 29, 72)");
      expect(heading.style.backgroundColor).toBe("");
    });
  });

  describe("existing variants still work", () => {
    it("professional uses borderColor and color", () => {
      render(<SectionHeader {...defaultProps} variant="professional" />);
      const heading = screen.getByText("Experience");
      expect(heading.style.borderColor).toBe("rgb(225, 29, 72)");
      expect(heading.style.color).toBe("rgb(225, 29, 72)");
    });

    it("creative uses borderColor and color", () => {
      render(<SectionHeader {...defaultProps} variant="creative" />);
      const heading = screen.getByText("Experience");
      expect(heading.style.borderColor).toBe("rgb(225, 29, 72)");
      expect(heading.style.color).toBe("rgb(225, 29, 72)");
    });

    it("modern uses borderColor and color", () => {
      render(<SectionHeader {...defaultProps} variant="modern" />);
      const heading = screen.getByText("Experience");
      expect(heading.style.borderColor).toBe("rgb(225, 29, 72)");
      expect(heading.style.color).toBe("rgb(225, 29, 72)");
    });
  });

  // ---------------------------------------------------------------------------
  // textColor prop
  // ---------------------------------------------------------------------------

  describe("textColor prop", () => {
    const textColor = "#334155";

    describe("border-based variants (professional, modern, minimal, executive, creative, compact, elegant, iconic, accent, timeline)", () => {
      const borderVariants = [
        "professional",
        "modern",
        "minimal",
        "executive",
        "creative",
        "compact",
        "elegant",
        "iconic",
        "accent",
        "timeline",
      ] as const;

      borderVariants.forEach((variant) => {
        it(`${variant}: heading text uses textColor instead of color`, () => {
          render(
            <SectionHeader
              {...defaultProps}
              variant={variant}
              textColor={textColor}
            />,
          );
          const heading = screen.getByText("Experience");
          expect(heading.style.color).toBe("rgb(51, 65, 85)");
          // border still uses primary color
          expect(heading.style.borderColor).toBe("rgb(225, 29, 72)");
        });
      });
    });

    describe("background-based variants (bold, vivid)", () => {
      const bgVariants = ["bold", "vivid"] as const;

      bgVariants.forEach((variant) => {
        it(`${variant}: heading text uses textColor when different from color`, () => {
          render(
            <SectionHeader
              {...defaultProps}
              variant={variant}
              textColor={textColor}
            />,
          );
          const heading = screen.getByText("Experience");
          expect(heading.style.color).toBe("rgb(51, 65, 85)");
          // background still uses primary color
          expect(heading.style.backgroundColor).toBe("rgb(225, 29, 72)");
        });

        it(`${variant}: heading text defaults to white when textColor equals color`, () => {
          render(
            <SectionHeader
              {...defaultProps}
              variant={variant}
              textColor="#e11d48"
            />,
          );
          const heading = screen.getByText("Experience");
          expect(heading.style.color).toBe("rgb(255, 255, 255)");
        });

        it(`${variant}: heading text defaults to white when textColor is not provided`, () => {
          render(<SectionHeader {...defaultProps} variant={variant} />);
          const heading = screen.getByText("Experience");
          expect(heading.style.color).toBe("rgb(255, 255, 255)");
        });
      });
    });

    it("falls back to color when textColor is not provided (non-bg variant)", () => {
      render(<SectionHeader {...defaultProps} variant="professional" />);
      const heading = screen.getByText("Experience");
      expect(heading.style.color).toBe("rgb(225, 29, 72)");
    });
  });
});
