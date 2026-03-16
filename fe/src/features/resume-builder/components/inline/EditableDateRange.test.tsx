import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditableDateRange } from "./EditableDateRange";

describe("EditableDateRange", () => {
  const defaultProps = {
    startDate: "2020-01-15",
    endDate: "2023-06-30",
    isCurrent: false,
    onStartDateChange: vi.fn(),
    onEndDateChange: vi.fn(),
    onIsCurrentChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders formatted date range in display mode", () => {
    render(<EditableDateRange {...defaultProps} />);
    // formatDate formats as "Mon Year"
    const text = screen.getByText(/Jan 2020/);
    expect(text).toBeInTheDocument();
  });

  it("shows 'Present' when isCurrent is true", () => {
    render(<EditableDateRange {...defaultProps} isCurrent={true} />);
    expect(screen.getByText(/Present/)).toBeInTheDocument();
  });

  it("shows custom currentLabel", () => {
    render(
      <EditableDateRange
        {...defaultProps}
        isCurrent={true}
        currentLabel="Current"
      />,
    );
    expect(screen.getByText(/Current/)).toBeInTheDocument();
  });

  it("shows 'Add dates' when no dates are set", () => {
    render(
      <EditableDateRange
        {...defaultProps}
        startDate=""
        endDate=""
      />,
    );
    expect(screen.getByText("Add dates")).toBeInTheDocument();
  });

  it("opens popover on click", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    expect(screen.getByText("Start Date")).toBeInTheDocument();
    expect(screen.getByText("End Date")).toBeInTheDocument();
  });

  it("calls onStartDateChange when start date changes", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    const startInput = screen.getByDisplayValue("2020-01-15");
    fireEvent.change(startInput, { target: { value: "2021-03-01" } });

    expect(defaultProps.onStartDateChange).toHaveBeenCalledWith("2021-03-01");
  });

  it("calls onEndDateChange when end date changes", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    const endInput = screen.getByDisplayValue("2023-06-30");
    fireEvent.change(endInput, { target: { value: "2024-01-01" } });

    expect(defaultProps.onEndDateChange).toHaveBeenCalledWith("2024-01-01");
  });

  it("calls onIsCurrentChange when checkbox is toggled", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    const checkbox = screen.getByRole("checkbox");
    await user.click(checkbox);

    expect(defaultProps.onIsCurrentChange).toHaveBeenCalledWith(true);
    expect(defaultProps.onEndDateChange).toHaveBeenCalledWith("");
  });

  it("hides end date input when isCurrent is true", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} isCurrent={true} />);

    await user.click(screen.getByText(/Present/));

    expect(screen.queryByText("End Date")).not.toBeInTheDocument();
  });

  it("does not open popover when not editable", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} editable={false} />);

    await user.click(screen.getByText(/Jan 2020/));

    expect(screen.queryByText("Start Date")).not.toBeInTheDocument();
  });
});
