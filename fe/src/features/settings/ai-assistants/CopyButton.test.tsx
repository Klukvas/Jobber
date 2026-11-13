import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CopyButton } from "./CopyButton";

vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("CopyButton keyboard fallback", () => {
  it.each(["unavailable", "rejects"])("focuses and selects the readonly value when clipboard %s", async (failure) => {
    const user = userEvent.setup();
    if (failure === "unavailable") {
      vi.spyOn(navigator, "clipboard", "get").mockReturnValue(undefined as unknown as Clipboard);
    } else {
      vi.spyOn(navigator.clipboard, "writeText").mockRejectedValue(new Error("Denied"));
    }
    const value = "first line\nsecond line";
    render(<CopyButton label="Copy setup" value={value} />);
    await user.tab();
    expect(screen.getByRole("button", { name: "Copy setup" })).toHaveFocus();
    await user.keyboard("{Enter}");
    const field = await screen.findByRole("textbox", { name: "Copy setup" });
    expect(field).toHaveAttribute("readonly");
    expect(field).toHaveValue(value);
    expect(field).toHaveFocus();
    expect(field).toHaveProperty("selectionStart", 0);
    expect(field).toHaveProperty("selectionEnd", value.length);
    expect(screen.getByRole("status")).toHaveTextContent("settings.aiAssistants.copyError");
  });
});
