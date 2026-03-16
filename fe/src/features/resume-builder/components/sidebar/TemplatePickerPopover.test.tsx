import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TemplatePickerPopover } from "./TemplatePickerPopover";
import { createMockStoreState } from "../__tests__/testHelpers";

const mockState = createMockStoreState();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

vi.mock("./TemplateThumbnail", () => ({
  TemplateThumbnail: ({ templateId }: { templateId: string; width: number }) => (
    <div data-testid={`thumbnail-${templateId}`}>Thumbnail</div>
  ),
}));

vi.mock("../../lib/templateRegistry", () => ({
  TEMPLATE_LIST: [
    {
      id: "00000000-0000-0000-0000-000000000001",
      nameKey: "resumeBuilder.templates.professional",
    },
    {
      id: "00000000-0000-0000-0000-000000000002",
      nameKey: "resumeBuilder.templates.modern",
    },
  ],
}));

describe("TemplatePickerPopover", () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders nothing when isOpen is false", () => {
    const { container } = render(
      <TemplatePickerPopover isOpen={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the dialog when isOpen is true", () => {
    render(<TemplatePickerPopover {...defaultProps} />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("renders the heading", () => {
    render(<TemplatePickerPopover {...defaultProps} />);
    expect(
      screen.getByText("resumeBuilder.design.template"),
    ).toBeInTheDocument();
  });

  it("renders template buttons", () => {
    render(<TemplatePickerPopover {...defaultProps} />);
    expect(
      screen.getByText("resumeBuilder.templates.professional"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.templates.modern"),
    ).toBeInTheDocument();
  });

  it("renders thumbnails for each template", () => {
    render(<TemplatePickerPopover {...defaultProps} />);
    expect(
      screen.getByTestId("thumbnail-00000000-0000-0000-0000-000000000001"),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId("thumbnail-00000000-0000-0000-0000-000000000002"),
    ).toBeInTheDocument();
  });

  it("calls updateDesign and onClose when a template is selected", async () => {
    const user = userEvent.setup();
    render(<TemplatePickerPopover {...defaultProps} />);

    await user.click(
      screen.getByText("resumeBuilder.templates.modern"),
    );

    expect(mockState.updateDesign).toHaveBeenCalledWith({
      template_id: "00000000-0000-0000-0000-000000000002",
    });
    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it("calls onClose when close button is clicked", async () => {
    const user = userEvent.setup();
    render(<TemplatePickerPopover {...defaultProps} />);

    const closeBtn = screen.getByLabelText("common.close");
    await user.click(closeBtn);

    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it("calls onClose on Escape key", async () => {
    const user = userEvent.setup();
    render(<TemplatePickerPopover {...defaultProps} />);

    await user.keyboard("{Escape}");

    expect(defaultProps.onClose).toHaveBeenCalled();
  });
});
