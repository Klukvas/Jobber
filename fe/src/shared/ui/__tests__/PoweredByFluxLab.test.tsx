import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { PoweredByFluxLab, FLUXLAB_URL } from "../PoweredByFluxLab";

// This attribution is a delivery requirement, so its exact wording, target and
// external-link safety are all asserted rather than assumed.
describe("PoweredByFluxLab", () => {
  it("reads exactly 'Powered by FluxLab'", () => {
    render(<PoweredByFluxLab />);
    expect(screen.getByText("Powered by FluxLab")).toBeInTheDocument();
  });

  it("is a real link to flux-lab.dev", () => {
    render(<PoweredByFluxLab />);
    const link = screen.getByRole("link", { name: "Powered by FluxLab" });

    expect(link).toHaveAttribute("href", "https://flux-lab.dev/en");
    expect(FLUXLAB_URL).toBe("https://flux-lab.dev/en");
  });

  it("opens in a new tab without handing over the opener", () => {
    render(<PoweredByFluxLab />);
    const link = screen.getByRole("link", { name: "Powered by FluxLab" });

    expect(link).toHaveAttribute("target", "_blank");
    expect(link.getAttribute("rel")).toContain("noopener");
    expect(link.getAttribute("rel")).toContain("noreferrer");
  });

  it("carries a visible hover and focus treatment", () => {
    render(<PoweredByFluxLab />);
    const link = screen.getByRole("link", { name: "Powered by FluxLab" });

    expect(link.className).toContain("hover:underline");
    expect(link.className).toContain("focus-visible:ring-2");
  });

  it("accepts the caller's colour scheme so both footers can host it", () => {
    render(<PoweredByFluxLab className="text-slate-600" />);
    const link = screen.getByRole("link", { name: "Powered by FluxLab" });

    expect(link.className).toContain("text-slate-600");
  });

  // Measured at 119x20 in the footer: wide enough, half the height a thumb
  // needs. The minimum lifts from `sm` up so the pointer footers keep their
  // line height.
  it("is at least 44px tall on phones and no taller from sm up", () => {
    render(<PoweredByFluxLab />);
    const link = screen.getByRole("link", { name: "Powered by FluxLab" });

    expect(link.className).toContain("min-h-11");
    expect(link.className).toContain("sm:min-h-0");
  });

  // The attribution is a delivery requirement: growing the box must not have
  // moved the destination.
  it("still points at the exact required URL after the resize", () => {
    render(<PoweredByFluxLab className="min-h-0" />);
    const link = screen.getByRole("link", { name: "Powered by FluxLab" });

    expect(link).toHaveAttribute("href", "https://flux-lab.dev/en");
  });
});
