import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
import { ConsentLoading } from "./ConsentLoading";

afterEach(() => { cleanup(); vi.useRealTimers(); });

describe("consent loading", () => {
  it("reserves the layout immediately and delays visible skeletons by 200ms", () => {
    vi.useFakeTimers();
    render(<ConsentLoading />);
    const skeleton = screen.getByRole("status", { name: "common.loading" }).firstElementChild;
    expect(skeleton).toHaveClass("invisible");
    act(() => vi.advanceTimersByTime(199));
    expect(skeleton).toHaveClass("invisible");
    act(() => vi.advanceTimersByTime(1));
    expect(skeleton).not.toHaveClass("invisible");
  });
});
