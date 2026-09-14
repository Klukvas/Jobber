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
    expect(
      screen.getByRole("button", { name: "Click me" }),
    ).toBeInTheDocument();
  });

  it.each([
    "default",
    "destructive",
    "outline",
    "secondary",
    "ghost",
    "link",
  ] as const)("renders variant=%s without crashing", (variant) => {
    render(<Button variant={variant}>btn</Button>);
    expect(screen.getByRole("button", { name: "btn" })).toBeInTheDocument();
  });

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

  /**
   * The size scale was drawn for a pointer — `default` 40px tall, `sm` 36,
   * `icon` 40x40 — and every one of those is the same control a thumb gets on
   * a phone. Measured on a real device: "Back to jobs" 137x40, "Add comment"
   * 152x36, the header menu and theme toggles 40x40, the onboarding Skip/Next
   * pair 36 tall. Asserted on the classes, because jsdom has no layout to
   * measure and the whole point is that the rule is media-scoped.
   */
  describe("tap targets on phones", () => {
    it.each(["default", "sm"] as const)(
      "gives size=%s a 44px minimum below sm",
      (size) => {
        render(<Button size={size}>sz</Button>);
        expect(screen.getByRole("button").className).toContain("max-sm:h-11");
      },
    );

    it("gives the icon size a 44x44 box below sm", () => {
      render(<Button size="icon">x</Button>);
      const className = screen.getByRole("button").className;

      expect(className).toContain("max-sm:h-11");
      expect(className).toContain("max-sm:w-11");
    });

    it("leaves the pointer sizes exactly as they were", () => {
      const { rerender } = render(<Button>sz</Button>);
      expect(screen.getByRole("button").className).toContain("h-10");

      rerender(<Button size="sm">sz</Button>);
      expect(screen.getByRole("button").className).toContain("h-9");

      rerender(<Button size="icon">sz</Button>);
      expect(screen.getByRole("button").className).toContain("h-10 w-10");
    });

    // `lg` is already 44.
    it("does not double up on a size that is already big enough", () => {
      render(
        <Button size="lg">sz</Button>,
      );
      const className = screen.getByRole("button").className;

      expect(className).toContain("h-11");
      expect(className).not.toContain("max-sm:h-11");
    });

    // A link-styled button is inline text inside a sentence; a 44px box there
    // would open a hole in the paragraph.
    it.each(["default", "sm", "icon"] as const)(
      "leaves the link variant inline at size=%s",
      (size) => {
        render(
          <Button variant="link" size={size}>
            sz
          </Button>,
        );
        expect(screen.getByRole("button").className).not.toContain(
          "max-sm:h-11",
        );
      },
    );
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

  /**
   * 40px is a pointer height, and it was the height everywhere — the settings
   * name and email fields measured 40 on a 390px screen, under the 44 WCAG
   * 2.5.5 asks of a target a thumb has to hit. Phones only, so desktop forms
   * keep the density they were drawn with; the button scale solves the same
   * problem the same way. Asserted on classes: jsdom has no layout.
   */
  it("grows to a 44px target on phones", () => {
    render(<Input data-testid="inp" />);
    expect(screen.getByTestId("inp").className).toContain("max-sm:h-11");
  });

  it("keeps its drawn height on a pointer layout", () => {
    render(<Input data-testid="inp" />);
    expect(screen.getByTestId("inp").className).toMatch(/(^|\s)h-10(\s|$)/);
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
    render(<EmptyState title="Empty" action={<button>Add</button>} />);
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
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
