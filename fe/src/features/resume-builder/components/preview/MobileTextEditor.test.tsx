import { beforeEach, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MobileTextEditor } from "./MobileTextEditor";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { createMockResume } from "../__tests__/testHelpers";
vi.mock("../../hooks/useSectionPersistence", () => ({ useSectionPersistence: () => ({ add: vi.fn(), remove: vi.fn() }) }));
beforeEach(() => {
  useResumeBuilderStore.getState().setResume(createMockResume({
    layout_mode: "double-left", skill_display: "circle", template_id: "modern",
    section_order: [{ section_key: "summary", sort_order: 0, is_visible: true, column: "main" }, { section_key: "experience", sort_order: 1, is_visible: true, column: "main" }],
  }));
});
it("uses the existing inline fields and Escape while leaving saved design untouched", async () => {
  const user = userEvent.setup();
  const original = useResumeBuilderStore.getState().resume;
  render(<MobileTextEditor />);
  await user.click(screen.getByRole("textbox", { name: "Your Name" }));
  const input = screen.getByRole("textbox", { name: "Your Name" });
  await user.clear(input); await user.type(input, "Synthetic draft");
  expect(useResumeBuilderStore.getState().resume?.contact?.full_name).toBe("Synthetic draft");
  await user.keyboard("{Escape}");
  expect(useResumeBuilderStore.getState().resume?.contact?.full_name).toBe(original?.contact?.full_name);
  expect(useResumeBuilderStore.getState().resume?.template_id).toBe("modern");
  expect(useResumeBuilderStore.getState().resume?.layout_mode).toBe("double-left");
  expect(useResumeBuilderStore.getState().resume?.skill_display).toBe("circle");
  expect(useResumeBuilderStore.getState().resume?.section_order).toEqual(original?.section_order);
});
