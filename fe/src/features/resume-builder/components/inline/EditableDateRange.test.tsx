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
    expect(screen.getByText(/resumeBuilder.inline.present/)).toBeInTheDocument();
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
    render(<EditableDateRange {...defaultProps} startDate="" endDate="" />);
    expect(screen.getByText("resumeBuilder.inline.addDates")).toBeInTheDocument();
  });

  it("opens popover on click", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    expect(screen.getByText("resumeBuilder.inline.startDate")).toBeInTheDocument();
    expect(screen.getByText("resumeBuilder.inline.endDate")).toBeInTheDocument();
  });

  it("calls onStartDateChange when start date changes", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    const startInput = screen.getByDisplayValue("2020-01-15");
    fireEvent.change(startInput, { target: { value: "2021-03-01" } });

    expect(defaultProps.onStartDateChange).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "common.save" }));
    expect(defaultProps.onStartDateChange).toHaveBeenCalledWith("2021-03-01");
  });

  it("calls onEndDateChange when end date changes", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    const endInput = screen.getByDisplayValue("2023-06-30");
    fireEvent.change(endInput, { target: { value: "2024-01-01" } });

    await user.click(screen.getByRole("button", { name: "common.save" }));
    expect(defaultProps.onEndDateChange).toHaveBeenCalledWith("2024-01-01");
  });

  it("calls onIsCurrentChange when checkbox is toggled", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} />);

    await user.click(screen.getByText(/Jan 2020/));

    const checkbox = screen.getByRole("checkbox");
    await user.click(checkbox);

    await user.click(screen.getByRole("button", { name: "common.save" }));
    expect(defaultProps.onIsCurrentChange).toHaveBeenCalledWith(true);
    expect(defaultProps.onEndDateChange).toHaveBeenCalledWith("");
  });

  it("hides end date input when isCurrent is true", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} isCurrent={true} />);

    await user.click(screen.getByText(/resumeBuilder.inline.present/));

    expect(screen.queryByText("End Date")).not.toBeInTheDocument();
  });

  it("does not open popover when not editable", async () => {
    const user = userEvent.setup();
    render(<EditableDateRange {...defaultProps} editable={false} />);

    await user.click(screen.getByText(/Jan 2020/));

    expect(screen.queryByText("Start Date")).not.toBeInTheDocument();
  });
});

it("omits empty dates from read-only output", () => {
  const { container } = render(
    <EditableDateRange
      startDate=""
      endDate=""
      isCurrent={false}
      editable={false}
      onStartDateChange={vi.fn()}
      onEndDateChange={vi.fn()}
      onIsCurrentChange={vi.fn()}
    />,
  );
  expect(container).toBeEmptyDOMElement();
});
it("retains a partial date in read-only output", () => {
  render(
    <EditableDateRange
      startDate="2020-01-15"
      endDate=""
      isCurrent={false}
      editable={false}
      onStartDateChange={vi.fn()}
      onEndDateChange={vi.fn()}
      onIsCurrentChange={vi.fn()}
    />,
  );
  expect(screen.getByText("Jan 2020")).toBeInTheDocument();
});

it("keeps an invalid pair out of the document and commits a corrected pair atomically", async () => {
  const user = userEvent.setup();
  const onRangeChange = vi.fn();
  render(<EditableDateRange startDate="2025-01-01" endDate="2025-02-01" isCurrent={false} onStartDateChange={vi.fn()} onEndDateChange={vi.fn()} onIsCurrentChange={vi.fn()} onRangeChange={onRangeChange} />);
  await user.click(screen.getByRole("button"));
  fireEvent.change(screen.getByLabelText("resumeBuilder.inline.endDate"), { target: { value: "2024-01-01" } });
  expect(screen.getByRole("alert")).toHaveTextContent("resumeBuilder.inline.invalidDateRange");
  expect(screen.getByRole("button", { name: "common.save" })).toBeDisabled();
  expect(onRangeChange).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText("resumeBuilder.inline.endDate"), { target: { value: "2025-01-01" } });
  await user.click(screen.getByRole("button", { name: "common.save" }));
  expect(onRangeChange).toHaveBeenCalledExactlyOnceWith({ start_date: "2025-01-01", end_date: "2025-01-01", is_current: false });
});

it("allows repairing a legacy invalid range using current and clearing end date", async () => {
  const user = userEvent.setup();
  const onRangeChange = vi.fn();
  render(<EditableDateRange startDate="2025-01-01" endDate="2024-01-01" isCurrent={false} onStartDateChange={vi.fn()} onEndDateChange={vi.fn()} onIsCurrentChange={vi.fn()} onRangeChange={onRangeChange} />);
  await user.click(screen.getByRole("button"));
  expect(screen.getByRole("button", { name: "common.save" })).toBeDisabled();
  await user.click(screen.getByRole("checkbox"));
  await user.click(screen.getByRole("button", { name: "common.save" }));
  expect(onRangeChange).toHaveBeenCalledExactlyOnceWith({ start_date: "2025-01-01", end_date: "", is_current: true });
});
