import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { JobCommentsSection } from "../JobCommentsSection";
import type { CommentDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options?.preview ? `${key}:${String(options.preview)}` : key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));

const comment: CommentDTO = {
  id: "c-1",
  job_id: "job-1",
  content: "Recruiter said they would call back",
  created_at: "2026-01-01T10:00:00Z",
  updated_at: "2026-01-01T10:00:00Z",
};

function renderSection(overrides: Partial<Parameters<typeof JobCommentsSection>[0]> = {}) {
  const props = {
    comments: [comment],
    newComment: "",
    onChangeComment: vi.fn(),
    onAddComment: vi.fn(),
    isAdding: false,
    onUpdateComment: vi.fn().mockResolvedValue(undefined),
    isUpdating: false,
    onDeleteComment: vi.fn().mockResolvedValue(undefined),
    isDeleting: false,
    ...overrides,
  };
  render(<JobCommentsSection {...props} />);
  return props;
}

/** A mutation that never settles, so "still in flight" is observable. */
function pendingCall() {
  return vi.fn().mockReturnValue(new Promise(() => {}));
}

function rejectingCall() {
  return vi.fn().mockRejectedValue(new Error("the server said no"));
}

describe("JobCommentsSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("offers edit and delete on every comment", () => {
    renderSection();

    expect(screen.getByLabelText("jobs.editComment")).toBeInTheDocument();
    expect(screen.getByLabelText("jobs.deleteComment")).toBeInTheDocument();
  });

  it("saves an edited comment", async () => {
    const user = userEvent.setup();
    const props = renderSection();

    await user.click(screen.getByLabelText("jobs.editComment"));
    const textarea = screen.getByLabelText("jobs.editComment");
    await user.clear(textarea);
    await user.type(textarea, "They called back");
    await user.click(screen.getByRole("button", { name: /common\.save/ }));

    expect(props.onUpdateComment).toHaveBeenCalledWith(
      "c-1",
      "They called back",
    );
  });

  it("does not submit an unchanged edit", async () => {
    const user = userEvent.setup();
    const props = renderSection();

    await user.click(screen.getByLabelText("jobs.editComment"));
    await user.click(screen.getByRole("button", { name: /common\.save/ }));

    expect(props.onUpdateComment).not.toHaveBeenCalled();
  });

  it("restores the original text when an edit is cancelled", async () => {
    const user = userEvent.setup();
    const props = renderSection();

    await user.click(screen.getByLabelText("jobs.editComment"));
    await user.type(screen.getByLabelText("jobs.editComment"), " extra");
    await user.click(screen.getByRole("button", { name: /common\.cancel/ }));

    expect(props.onUpdateComment).not.toHaveBeenCalled();
    expect(
      screen.getByText("Recruiter said they would call back"),
    ).toBeInTheDocument();
  });

  it("never deletes without confirmation, and names the comment in the confirm", async () => {
    const user = userEvent.setup();
    const props = renderSection();

    await user.click(screen.getByLabelText("jobs.deleteComment"));

    expect(props.onDeleteComment).not.toHaveBeenCalled();
    expect(
      screen.getByText(/jobs\.deleteCommentConfirm:Recruiter said they would/),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /common\.delete/ }));
    expect(props.onDeleteComment).toHaveBeenCalledWith("c-1");
  });

  it("keeps the comment when the confirm is dismissed", async () => {
    const user = userEvent.setup();
    const props = renderSection();

    await user.click(screen.getByLabelText("jobs.deleteComment"));
    await user.click(screen.getByRole("button", { name: /common\.cancel/ }));

    expect(props.onDeleteComment).not.toHaveBeenCalled();
  });

  it("marks a comment that has been edited", () => {
    renderSection({
      comments: [{ ...comment, updated_at: "2026-01-02T10:00:00Z" }],
    });

    expect(screen.getByText(/jobs\.commentEdited/)).toBeInTheDocument();
  });

  it("does not mark an untouched comment as edited", () => {
    renderSection();

    expect(screen.queryByText(/jobs\.commentEdited/)).not.toBeInTheDocument();
  });

  /**
   * A 16px glyph in `p-1.5` is a 28x28 target — under the 44px WCAG 2.5.5 asks
   * for, on two controls a few pixels apart, one of which deletes a comment for
   * good. Asserted on the classes because jsdom has no layout to measure, the
   * same way the Dialog close button and the consent banner are checked.
   */
  describe("touch targets", () => {
    it.each(["jobs.editComment", "jobs.deleteComment"])(
      "gives %s a 44px tap target on phones",
      (label) => {
        renderSection();
        const button = screen.getByLabelText(label);

        expect(button.className).toContain("h-11");
        expect(button.className).toContain("w-11");
      },
    );

    it.each(["jobs.editComment", "jobs.deleteComment"])(
      "keeps %s at its compact desktop size from sm up",
      (label) => {
        renderSection();
        const button = screen.getByLabelText(label);

        expect(button.className).toContain("sm:h-7");
        expect(button.className).toContain("sm:w-7");
      },
    );

    // The glyph is centred in the larger box rather than pinned to a corner, so
    // enlarging the target does not move or magnify what is drawn.
    it.each(["jobs.editComment", "jobs.deleteComment"])(
      "centres the unchanged glyph inside %s",
      (label) => {
        renderSection();
        const button = screen.getByLabelText(label);

        expect(button.className).toContain("items-center");
        expect(button.className).toContain("justify-center");
        const glyph = button.querySelector("svg");
        expect(glyph).toHaveClass("h-4", "w-4");
        expect(glyph).toHaveAttribute("aria-hidden");
      },
    );

    // Measured 152x36 on a phone: the shared Button at `sm`, which the size
    // scale's phone floor now lifts to 44 without touching the desktop 36.
    it("gives the submit button a 44px minimum on phones", () => {
      renderSection();
      const submit = screen.getByRole("button", { name: /jobs\.addComment/ });

      expect(submit.className).toContain("max-sm:h-11");
      expect(submit.className).toContain("h-9");
    });

    // Both controls share one constant, so the two can never drift apart.
    it("sizes both controls identically", () => {
      renderSection();
      const edit = screen.getByLabelText("jobs.editComment");
      const remove = screen.getByLabelText("jobs.deleteComment");

      const sizing = (el: HTMLElement) =>
        el.className
          .split(/\s+/)
          .filter((c) => /^(sm:)?[hw]-/.test(c))
          .sort();

      expect(sizing(edit)).toEqual(sizing(remove));
      expect(sizing(edit)).not.toHaveLength(0);
    });
  });

  /**
   * The editor and the delete confirm used to close the moment the request was
   * sent. A rejected PATCH therefore left an error toast, the original text
   * back on screen and the customer's rewritten comment gone — with nothing to
   * retry from. Both now close on the *resolution*, not on the send.
   */
  describe("a failed edit keeps the draft", () => {
    async function startEdit(props: ReturnType<typeof renderSection>) {
      const user = userEvent.setup();
      await user.click(screen.getByLabelText("jobs.editComment"));
      const textarea = screen.getByLabelText("jobs.editComment");
      await user.clear(textarea);
      await user.type(textarea, "They called back on Tuesday");
      await user.click(screen.getByRole("button", { name: /common\.save/ }));
      return { user, props };
    }

    it("closes the editor when the save succeeds", async () => {
      const props = renderSection();

      await startEdit(props);

      await waitFor(() =>
        expect(
          screen.queryByRole("button", { name: /common\.save/ }),
        ).not.toBeInTheDocument(),
      );
      expect(props.onUpdateComment).toHaveBeenCalledWith(
        "c-1",
        "They called back on Tuesday",
      );
    });

    it("keeps the editor open with the text still in it when the save fails", async () => {
      const props = renderSection({ onUpdateComment: rejectingCall() });

      await startEdit(props);

      await waitFor(() =>
        expect(props.onUpdateComment).toHaveBeenCalledTimes(1),
      );
      expect(screen.getByLabelText("jobs.editComment")).toHaveValue(
        "They called back on Tuesday",
      );
    });

    it("lets the failed save be retried without retyping", async () => {
      const onUpdateComment = rejectingCall();
      const props = renderSection({ onUpdateComment });
      const { user } = await startEdit(props);
      await waitFor(() => expect(onUpdateComment).toHaveBeenCalledTimes(1));

      await user.click(screen.getByRole("button", { name: /common\.save/ }));

      await waitFor(() => expect(onUpdateComment).toHaveBeenCalledTimes(2));
      expect(onUpdateComment).toHaveBeenLastCalledWith(
        "c-1",
        "They called back on Tuesday",
      );
    });

    // isUpdating disables both buttons, so one click cannot become two
    // requests for the same edit.
    it("disables save and cancel while the edit is in flight", async () => {
      const user = userEvent.setup();
      const props = {
        comments: [comment],
        newComment: "",
        onChangeComment: vi.fn(),
        onAddComment: vi.fn(),
        isAdding: false,
        onUpdateComment: pendingCall(),
        isUpdating: true,
        onDeleteComment: vi.fn().mockResolvedValue(undefined),
        isDeleting: false,
      };
      render(<JobCommentsSection {...props} />);

      await user.click(screen.getByLabelText("jobs.editComment"));

      expect(screen.getByRole("button", { name: /common\.save/ })).toBeDisabled();
      expect(
        screen.getByRole("button", { name: /common\.cancel/ }),
      ).toBeDisabled();
    });
  });

  describe("a failed delete keeps the confirmation", () => {
    async function confirmDelete() {
      const user = userEvent.setup();
      await user.click(screen.getByLabelText("jobs.deleteComment"));
      await user.click(screen.getByRole("button", { name: /common\.delete/ }));
      return user;
    }

    it("closes the confirm when the delete succeeds", async () => {
      const props = renderSection();

      await confirmDelete();

      await waitFor(() =>
        expect(
          screen.queryByRole("button", { name: /common\.delete/ }),
        ).not.toBeInTheDocument(),
      );
      expect(props.onDeleteComment).toHaveBeenCalledWith("c-1");
    });

    it("keeps the confirm open when the delete fails", async () => {
      const onDeleteComment = rejectingCall();
      renderSection({ onDeleteComment });

      await confirmDelete();

      await waitFor(() => expect(onDeleteComment).toHaveBeenCalledTimes(1));
      expect(
        screen.getByRole("button", { name: /common\.delete/ }),
      ).toBeInTheDocument();
    });

    it("sends exactly one request per confirmation", async () => {
      const onDeleteComment = rejectingCall();
      renderSection({ onDeleteComment });

      await confirmDelete();

      await waitFor(() => expect(onDeleteComment).toHaveBeenCalledTimes(1));
      expect(onDeleteComment).toHaveBeenCalledTimes(1);
    });

    // A delete in flight must not be dismissable: the dialog is the only thing
    // saying what is happening to the comment.
    it("disables the confirm's own controls while it is running", async () => {
      const user = userEvent.setup();
      const props = {
        comments: [comment],
        newComment: "",
        onChangeComment: vi.fn(),
        onAddComment: vi.fn(),
        isAdding: false,
        onUpdateComment: vi.fn().mockResolvedValue(undefined),
        isUpdating: false,
        onDeleteComment: pendingCall(),
        isDeleting: true,
      };
      render(<JobCommentsSection {...props} />);

      await user.click(screen.getByLabelText("jobs.deleteComment"));

      expect(
        screen.getByRole("button", { name: /common\.deleting/ }),
      ).toBeDisabled();
      expect(
        screen.getByRole("button", { name: /common\.cancel/ }),
      ).toBeDisabled();
      // ...and the dismiss affordance is gone entirely.
      expect(screen.queryByLabelText("common.close")).not.toBeInTheDocument();
    });
  });
});
