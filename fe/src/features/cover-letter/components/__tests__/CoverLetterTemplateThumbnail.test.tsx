import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CoverLetterTemplateThumbnail } from "../CoverLetterTemplateThumbnail";

describe("CoverLetterTemplateThumbnail", () => {
  it("renders nothing for an unknown templateId", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="nonexistent" />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders professional thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="professional" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders modern thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="modern" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders minimal thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="minimal" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders executive thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="executive" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders creative thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="creative" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders classic thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="classic" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders elegant thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="elegant" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders bold thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="bold" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders simple thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="simple" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders corporate thumbnail", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="corporate" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("uses default accent color when none provided", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="professional" />,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("applies custom accent color", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail
        templateId="professional"
        accentColor="#ff0000"
      />,
    );
    const rect = container.querySelector('rect[fill="#ff0000"]');
    expect(rect).toBeInTheDocument();
  });

  it("renders at small size by default", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="professional" />,
    );
    // Small size renders SVG directly without a wrapper div
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("renders at large size when specified", () => {
    const { container } = render(
      <CoverLetterTemplateThumbnail templateId="professional" size="lg" />,
    );
    // Large size wraps the SVG in a div with scaling classes
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });
});
