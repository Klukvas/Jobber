import { describe, expect, it } from "vitest";
import shell from "../../../index.html?raw";

describe("the page shell", () => {
  // A request to Google Fonts sends the visitor's address to Google before the
  // cookie banner has been answered.
  it("loads nothing from Google Fonts", () => {
    expect(shell).not.toMatch(/fonts\.(googleapis|gstatic)\.com/);
  });
});
