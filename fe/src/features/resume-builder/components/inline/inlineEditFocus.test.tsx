import { describe, it, expect, vi } from "vitest";
import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { EditableField } from "./EditableField";
import { EditableTextarea } from "./EditableTextarea";

/**
 * Clicking an inline field opened its input and left the keyboard behind.
 * Focus was taken from a `requestAnimationFrame` callback holding the node it
 * had been handed, so it landed a frame late — and, when committing the field
 * being left re-rendered the preview, on a node that no longer existed.
 * `document.activeElement` stayed on `<body>` or on the field just left, and
 * what the visitor typed next was lost or went into the wrong field.
 *
 * These assert on `document.activeElement` straight after the click, because
 * that is the moment the visitor starts typing. `userEvent.type` focuses the
 * element it is given, so any test that types into a field it looked up by
 * hand would pass with the bug still in place.
 */
describe("inline edit focus", () => {
  /**
   * Two fields whose text lives above them, re-keyed on every commit.
   *
   * The resume preview rebuilds when the store is written to, which is what
   * replaced the node the old focus callback was holding.
   */
  function TwoFields() {
    const [name, setName] = useState("John Doe");
    const [title, setTitle] = useState("Engineer");
    const [revision, setRevision] = useState(0);
    const commit = (set: (value: string) => void) => (value: string) => {
      set(value);
      setRevision((current) => current + 1);
    };

    return (
      <div key={revision}>
        <EditableField
          value={name}
          onChange={commit(setName)}
          placeholder="Name"
        />
        <EditableField
          value={title}
          onChange={commit(setTitle)}
          placeholder="Title"
        />
      </div>
    );
  }

  it("focuses the input a click opens", async () => {
    const user = userEvent.setup();
    render(
      <EditableField value="John Doe" onChange={() => {}} placeholder="Name" />,
    );

    await user.click(screen.getByText("John Doe"));

    const input = screen.getByRole("textbox");
    expect(document.activeElement).toBe(input);
  });

  it("leaves the caret at the end of the text", async () => {
    const user = userEvent.setup();
    render(
      <EditableField value="John Doe" onChange={() => {}} placeholder="Name" />,
    );

    await user.click(screen.getByText("John Doe"));

    const input = screen.getByRole("textbox") as HTMLInputElement;
    expect(input.selectionStart).toBe("John Doe".length);
    expect(input.selectionEnd).toBe("John Doe".length);
  });

  it("takes what is typed straight after the click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <EditableField value="John" onChange={onChange} placeholder="Name" />,
    );

    await user.click(screen.getByText("John"));
    // Typed at the keyboard, not into a node looked up by hand — this is what
    // the visitor actually does, and what used to go nowhere.
    await user.keyboard(" Doe{Enter}");

    expect(onChange).toHaveBeenCalledWith("John Doe");
  });

  it("moves the keyboard when a click switches fields", async () => {
    const user = userEvent.setup();
    render(<TwoFields />);

    await user.click(screen.getByText("John Doe"));
    expect(document.activeElement).toBe(screen.getByDisplayValue("John Doe"));

    await user.click(screen.getByText("Engineer"));

    const second = screen.getByDisplayValue("Engineer");
    expect(document.activeElement).toBe(second);
    // The field just left is back to display mode, so nothing can be typed
    // into it by mistake.
    expect(screen.queryByDisplayValue("John Doe")).not.toBeInTheDocument();
  });

  it("types into the field that was switched to, not the one left", async () => {
    const user = userEvent.setup();
    render(<TwoFields />);

    await user.click(screen.getByText("John Doe"));
    await user.click(screen.getByText("Engineer"));
    await user.keyboard("!");

    expect(screen.getByDisplayValue("Engineer!")).toBeInTheDocument();
  });

  it("focuses an empty field opened from its placeholder", async () => {
    const user = userEvent.setup();
    render(<EditableField value="" onChange={() => {}} placeholder="Name" />);

    await user.click(screen.getByText("Name"));

    expect(document.activeElement).toBe(screen.getByRole("textbox"));
  });

  it("focuses a field the keyboard opened", async () => {
    const user = userEvent.setup();
    render(
      <EditableField value="John Doe" onChange={() => {}} placeholder="Name" />,
    );

    screen.getByRole("textbox").focus();
    await user.keyboard("{Enter}");

    expect(document.activeElement).toBe(screen.getByDisplayValue("John Doe"));
  });

  // `date` and `email` inputs have no text selection to place a caret in, and
  // asking for one throws.
  it.each(["date", "email"] as const)("focuses a %s field", async (type) => {
    const user = userEvent.setup();
    render(
      <EditableField
        value=""
        type={type}
        onChange={() => {}}
        placeholder="When"
      />,
    );

    await user.click(screen.getByText("When"));

    expect(document.activeElement).toBe(
      document.querySelector(`input[type="${type}"]`),
    );
  });

  it("focuses a textarea a click opens", async () => {
    const user = userEvent.setup();
    render(
      <EditableTextarea
        value="A summary"
        onChange={() => {}}
        placeholder="Summary"
      />,
    );

    await user.click(screen.getByText("A summary"));

    const textarea = screen.getByRole("textbox");
    expect(textarea.tagName).toBe("TEXTAREA");
    expect(document.activeElement).toBe(textarea);
    expect((textarea as HTMLTextAreaElement).selectionStart).toBe(
      "A summary".length,
    );
  });

  it("takes what is typed into a textarea straight after the click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <EditableTextarea
        value="A summary"
        onChange={onChange}
        placeholder="Summary"
      />,
    );

    await user.click(screen.getByText("A summary"));
    await user.keyboard(" of me");
    await user.tab();

    expect(onChange).toHaveBeenCalledWith("A summary of me");
  });
});
