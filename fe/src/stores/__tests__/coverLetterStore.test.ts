import { describe, it, expect, beforeEach } from "vitest";
import { act } from "@testing-library/react";
import { useCoverLetterStore } from "../coverLetterStore";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

function createMockCoverLetter(
  overrides?: Partial<CoverLetterDTO>,
): CoverLetterDTO {
  return {
    id: "cl-1",
    resume_builder_id: null,
    job_id: null,
    title: "My Cover Letter",
    template: "professional",
    recipient_name: "Jane Smith",
    recipient_title: "Hiring Manager",
    company_name: "Acme Corp",
    company_address: "123 Main St",
    greeting: "Dear Jane Smith,",
    paragraphs: ["First paragraph", "Second paragraph"],
    closing: "Sincerely,",
    font_family: "Georgia",
    font_size: 12,
    primary_color: "#2563eb",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("coverLetterStore", () => {
  beforeEach(() => {
    act(() => {
      useCoverLetterStore.setState({
        coverLetter: null,
        isDirty: false,
        saveStatus: "idle",
      });
      useCoverLetterStore.temporal.getState().clear();
    });
  });

  describe("setCoverLetter", () => {
    it("sets cover letter and resets dirty and saveStatus", () => {
      const letter = createMockCoverLetter();

      act(() => {
        useCoverLetterStore.getState().markDirty();
        useCoverLetterStore.getState().setSaveStatus("saving");
      });

      expect(useCoverLetterStore.getState().isDirty).toBe(true);
      expect(useCoverLetterStore.getState().saveStatus).toBe("saving");

      act(() => {
        useCoverLetterStore.getState().setCoverLetter(letter);
      });

      const state = useCoverLetterStore.getState();
      expect(state.coverLetter).toEqual(letter);
      expect(state.isDirty).toBe(false);
      expect(state.saveStatus).toBe("idle");
    });
  });

  describe("updateField", () => {
    it("updates a single field and marks dirty", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(createMockCoverLetter());
      });

      act(() => {
        useCoverLetterStore.getState().updateField("title", "Updated Title");
      });

      const state = useCoverLetterStore.getState();
      expect(state.coverLetter?.title).toBe("Updated Title");
      expect(state.isDirty).toBe(true);
    });

    it("does nothing when coverLetter is null", () => {
      act(() => {
        useCoverLetterStore.getState().updateField("title", "Updated Title");
      });

      expect(useCoverLetterStore.getState().coverLetter).toBeNull();
    });
  });

  describe("updateFields", () => {
    it("updates multiple fields including job_id and marks dirty", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(createMockCoverLetter());
      });

      act(() => {
        useCoverLetterStore.getState().updateFields({
          company_name: "New Corp",
          job_id: "job-42",
          font_family: "Arial",
        });
      });

      const state = useCoverLetterStore.getState();
      expect(state.coverLetter?.company_name).toBe("New Corp");
      expect(state.coverLetter?.job_id).toBe("job-42");
      expect(state.coverLetter?.font_family).toBe("Arial");
      expect(state.isDirty).toBe(true);
    });

    it("does nothing when coverLetter is null", () => {
      act(() => {
        useCoverLetterStore.getState().updateFields({ title: "Nope" });
      });

      expect(useCoverLetterStore.getState().coverLetter).toBeNull();
    });
  });

  describe("addParagraph", () => {
    it("appends an empty string to paragraphs", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(
            createMockCoverLetter({ paragraphs: ["Existing"] }),
          );
      });

      act(() => {
        useCoverLetterStore.getState().addParagraph();
      });

      const state = useCoverLetterStore.getState();
      expect(state.coverLetter?.paragraphs).toEqual(["Existing", ""]);
      expect(state.isDirty).toBe(true);
    });

    it("does nothing when coverLetter is null", () => {
      act(() => {
        useCoverLetterStore.getState().addParagraph();
      });

      expect(useCoverLetterStore.getState().coverLetter).toBeNull();
    });
  });

  describe("updateParagraph", () => {
    it("updates paragraph at a specific index", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(
            createMockCoverLetter({ paragraphs: ["A", "B", "C"] }),
          );
      });

      act(() => {
        useCoverLetterStore.getState().updateParagraph(1, "Updated B");
      });

      const state = useCoverLetterStore.getState();
      expect(state.coverLetter?.paragraphs).toEqual(["A", "Updated B", "C"]);
      expect(state.isDirty).toBe(true);
    });
  });

  describe("removeParagraph", () => {
    it("removes paragraph at a specific index", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(
            createMockCoverLetter({ paragraphs: ["A", "B", "C"] }),
          );
      });

      act(() => {
        useCoverLetterStore.getState().removeParagraph(1);
      });

      const state = useCoverLetterStore.getState();
      expect(state.coverLetter?.paragraphs).toEqual(["A", "C"]);
      expect(state.isDirty).toBe(true);
    });
  });

  describe("markDirty / markClean", () => {
    it("markDirty sets isDirty to true and saveStatus to idle", () => {
      act(() => {
        useCoverLetterStore.getState().setSaveStatus("saved");
        useCoverLetterStore.getState().markDirty();
      });

      const state = useCoverLetterStore.getState();
      expect(state.isDirty).toBe(true);
      expect(state.saveStatus).toBe("idle");
    });

    it("markClean sets isDirty to false", () => {
      act(() => {
        useCoverLetterStore.getState().markDirty();
      });

      expect(useCoverLetterStore.getState().isDirty).toBe(true);

      act(() => {
        useCoverLetterStore.getState().markClean();
      });

      expect(useCoverLetterStore.getState().isDirty).toBe(false);
    });
  });

  describe("temporal middleware (undo/redo)", () => {
    it("undo reverts the last change", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(createMockCoverLetter({ title: "Original" }));
      });

      // Clear history after setCoverLetter so that's the baseline
      act(() => {
        useCoverLetterStore.temporal.getState().clear();
      });

      act(() => {
        useCoverLetterStore.getState().updateField("title", "Changed");
      });

      expect(useCoverLetterStore.getState().coverLetter?.title).toBe(
        "Changed",
      );

      act(() => {
        useCoverLetterStore.temporal.getState().undo();
      });

      expect(useCoverLetterStore.getState().coverLetter?.title).toBe(
        "Original",
      );
    });

    it("redo re-applies after undo", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(createMockCoverLetter({ title: "Original" }));
        useCoverLetterStore.temporal.getState().clear();
      });

      act(() => {
        useCoverLetterStore.getState().updateField("title", "Changed");
      });

      act(() => {
        useCoverLetterStore.temporal.getState().undo();
      });

      expect(useCoverLetterStore.getState().coverLetter?.title).toBe(
        "Original",
      );

      act(() => {
        useCoverLetterStore.temporal.getState().redo();
      });

      expect(useCoverLetterStore.getState().coverLetter?.title).toBe(
        "Changed",
      );
    });

    it("clear() resets history", () => {
      act(() => {
        useCoverLetterStore
          .getState()
          .setCoverLetter(createMockCoverLetter({ title: "Original" }));
        useCoverLetterStore.temporal.getState().clear();
      });

      act(() => {
        useCoverLetterStore.getState().updateField("title", "Changed");
      });

      expect(
        useCoverLetterStore.temporal.getState().pastStates.length,
      ).toBeGreaterThan(0);

      act(() => {
        useCoverLetterStore.temporal.getState().clear();
      });

      expect(useCoverLetterStore.temporal.getState().pastStates).toHaveLength(
        0,
      );
      expect(
        useCoverLetterStore.temporal.getState().futureStates,
      ).toHaveLength(0);
    });
  });
});
