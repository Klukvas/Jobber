import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { createRef } from "react";
import { Button } from "../Button";
import {
  Card,
  CardHeader,
  CardContent,
  CardFooter,
  CardTitle,
  CardDescription,
} from "../Card";
import { Input } from "../Input";
import { Label } from "../Label";
import { ErrorState } from "../ErrorState";
import { EmptyState } from "../EmptyState";
import { StatusBadge } from "../StatusBadge";
import {
  Skeleton,
  SkeletonCard,
  SkeletonList,
  SkeletonTable,
  SkeletonDetail,
} from "../Skeleton";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

// ---------- Button ----------
describe("Button", () => {
  it("renders with children", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole("button", { name: "Click me" })).toBeInTheDocument();
  });

  it.each(["default", "destructive", "outline", "secondary", "ghost", "link"] as const)(
    "renders variant=%s without crashing",
    (variant) => {
      render(<Button variant={variant}>btn</Button>);
      expect(screen.getByRole("button", { name: "btn" })).toBeInTheDocument();
    },
  );

  it.each(["default", "sm", "lg", "icon"] as const)(
    "renders size=%s without crashing",
    (size) => {
      render(<Button size={size}>sz</Button>);
      expect(screen.getByRole("button", { name: "sz" })).toBeInTheDocument();
    },
  );

  it("forwards ref", () => {
    const ref = createRef<HTMLButtonElement>();
    render(<Button ref={ref}>ref</Button>);
    expect(ref.current).toBeInstanceOf(HTMLButtonElement);
  });

  it("calls onClick handler", () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>click</Button>);
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("is disabled when disabled prop is set", () => {
    render(<Button disabled>no</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });
});

// ---------- Card ----------
describe("Card", () => {
  it("renders Card with children", () => {
    render(<Card data-testid="card">content</Card>);
    expect(screen.getByTestId("card")).toHaveTextContent("content");
  });

  it("renders CardHeader", () => {
    render(<CardHeader data-testid="header">hdr</CardHeader>);
    expect(screen.getByTestId("header")).toHaveTextContent("hdr");
  });

  it("renders CardTitle", () => {
    render(<CardTitle>Title</CardTitle>);
    expect(screen.getByText("Title")).toBeInTheDocument();
  });

  it("renders CardDescription", () => {
    render(<CardDescription>Desc</CardDescription>);
    expect(screen.getByText("Desc")).toBeInTheDocument();
  });

  it("renders CardContent", () => {
    render(<CardContent data-testid="content">body</CardContent>);
    expect(screen.getByTestId("content")).toHaveTextContent("body");
  });

  it("renders CardFooter", () => {
    render(<CardFooter data-testid="footer">foot</CardFooter>);
    expect(screen.getByTestId("footer")).toHaveTextContent("foot");
  });

  it("composes full card layout", () => {
    render(
      <Card data-testid="card">
        <CardHeader>
          <CardTitle>T</CardTitle>
          <CardDescription>D</CardDescription>
        </CardHeader>
        <CardContent>C</CardContent>
        <CardFooter>F</CardFooter>
      </Card>,
    );
    const card = screen.getByTestId("card");
    expect(card).toHaveTextContent("T");
    expect(card).toHaveTextContent("D");
    expect(card).toHaveTextContent("C");
    expect(card).toHaveTextContent("F");
  });

  it("forwards ref on Card", () => {
    const ref = createRef<HTMLDivElement>();
    render(<Card ref={ref}>r</Card>);
    expect(ref.current).toBeInstanceOf(HTMLDivElement);
  });
});

// ---------- Input ----------
describe("Input", () => {
  it("renders an input element", () => {
    render(<Input placeholder="enter" />);
    expect(screen.getByPlaceholderText("enter")).toBeInTheDocument();
  });

  it("handles onChange", () => {
    const onChange = vi.fn();
    render(<Input onChange={onChange} placeholder="type" />);
    fireEvent.change(screen.getByPlaceholderText("type"), {
      target: { value: "hello" },
    });
    expect(onChange).toHaveBeenCalledOnce();
  });

  it("forwards ref", () => {
    const ref = createRef<HTMLInputElement>();
    render(<Input ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLInputElement);
  });

  it("passes type prop", () => {
    render(<Input type="email" data-testid="inp" />);
    expect(screen.getByTestId("inp")).toHaveAttribute("type", "email");
  });

  it("is disabled when disabled prop is set", () => {
    render(<Input disabled data-testid="inp" />);
    expect(screen.getByTestId("inp")).toBeDisabled();
  });
});

// ---------- Label ----------
describe("Label", () => {
  it("renders with text", () => {
    render(<Label>Email</Label>);
    expect(screen.getByText("Email")).toBeInTheDocument();
  });

  it("associates with input via htmlFor", () => {
    render(
      <>
        <Label htmlFor="email-input">Email</Label>
        <Input id="email-input" />
      </>,
    );
    expect(screen.getByText("Email")).toHaveAttribute("for", "email-input");
  });

  it("forwards ref", () => {
    const ref = createRef<HTMLLabelElement>();
    render(<Label ref={ref}>L</Label>);
    expect(ref.current).toBeInstanceOf(HTMLLabelElement);
  });
});

// ---------- ErrorState ----------
describe("ErrorState", () => {
  it("renders message", () => {
    render(<ErrorState message="Something broke" />);
    expect(screen.getByText("Something broke")).toBeInTheDocument();
  });

  it("renders default title via translation key", () => {
    render(<ErrorState message="err" />);
    expect(screen.getByText("errors.somethingWentWrong")).toBeInTheDocument();
  });

  it("renders custom title", () => {
    render(<ErrorState title="Oops" message="err" />);
    expect(screen.getByText("Oops")).toBeInTheDocument();
  });

  it("renders retry button when onRetry is provided", () => {
    const onRetry = vi.fn();
    render(<ErrorState message="err" onRetry={onRetry} />);
    const btn = screen.getByRole("button", { name: "common.tryAgain" });
    expect(btn).toBeInTheDocument();
    fireEvent.click(btn);
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("does not render retry button when onRetry is not provided", () => {
    render(<ErrorState message="err" />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("has role=alert", () => {
    render(<ErrorState message="err" />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});

// ---------- EmptyState ----------
describe("EmptyState", () => {
  it("renders title", () => {
    render(<EmptyState title="No items" />);
    expect(screen.getByText("No items")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(<EmptyState title="No items" description="Add something" />);
    expect(screen.getByText("Add something")).toBeInTheDocument();
  });

  it("does not render description when omitted", () => {
    const { container } = render(<EmptyState title="No items" />);
    // Only the title paragraph should be present, no description paragraph
    const paragraphs = container.querySelectorAll("p");
    expect(paragraphs).toHaveLength(0);
  });

  it("renders custom icon", () => {
    render(
      <EmptyState
        title="Empty"
        icon={<span data-testid="custom-icon">IC</span>}
      />,
    );
    expect(screen.getByTestId("custom-icon")).toBeInTheDocument();
  });

  it("renders action when provided", () => {
    render(
      <EmptyState
        title="Empty"
        action={<button>Add</button>}
      />,
    );
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
  });
});

// ---------- StatusBadge ----------
describe("StatusBadge", () => {
  it.each(["active", "on_hold", "rejected", "offer", "archived"] as const)(
    "renders status=%s",
    (status) => {
      render(<StatusBadge status={status} />);
      // The translated key should be rendered as the text
      expect(screen.getByText(`applications.status${status.charAt(0).toUpperCase() + status.slice(1).replace(/_([a-z])/g, (_, c: string) => c.toUpperCase())}`)).toBeInTheDocument();
    },
  );

  it("renders unknown status as raw text", () => {
    render(<StatusBadge status={"unknown" as never} />);
    expect(screen.getByText("unknown")).toBeInTheDocument();
  });

  it("renders with small size", () => {
    const { container } = render(<StatusBadge status="active" size="sm" />);
    const span = container.querySelector("span");
    expect(span?.className).toContain("text-xs");
  });

  it("renders with large size", () => {
    const { container } = render(<StatusBadge status="active" size="lg" />);
    const span = container.querySelector("span");
    expect(span?.className).toContain("text-base");
  });
});

// ---------- Skeleton ----------
describe("Skeleton", () => {
  it("renders with animation class", () => {
    const { container } = render(<Skeleton />);
    const el = container.firstElementChild;
    expect(el?.className).toContain("animate-pulse");
  });

  it("has aria-hidden", () => {
    const { container } = render(<Skeleton />);
    expect(container.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });

  it("accepts custom className", () => {
    const { container } = render(<Skeleton className="h-4 w-full" />);
    expect(container.firstElementChild?.className).toContain("h-4");
  });
});

describe("SkeletonCard", () => {
  it("renders without crashing", () => {
    const { container } = render(<SkeletonCard />);
    expect(container.firstElementChild).toBeTruthy();
  });
});

describe("SkeletonList", () => {
  it("renders default 3 cards", () => {
    const { container } = render(<SkeletonList />);
    const cards = container.querySelectorAll(".rounded-lg");
    expect(cards.length).toBe(3);
  });

  it("renders custom count", () => {
    const { container } = render(<SkeletonList count={5} />);
    const cards = container.querySelectorAll(".rounded-lg");
    expect(cards.length).toBe(5);
  });
});

describe("SkeletonTable", () => {
  it("renders default rows and cols", () => {
    const { container } = render(<SkeletonTable />);
    const rows = container.querySelectorAll(".flex.gap-4");
    expect(rows.length).toBe(5);
  });
});

describe("SkeletonDetail", () => {
  it("renders without crashing", () => {
    const { container } = render(<SkeletonDetail />);
    expect(container.firstElementChild).toBeTruthy();
  });
});
