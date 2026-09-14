import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-[color,background-color,border-color,transform] duration-150 ease-out active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 disabled:active:scale-100 motion-reduce:transition-none motion-reduce:active:scale-100",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive:
          "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline:
          "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-10 px-4 py-2",
        sm: "h-9 rounded-md px-3",
        lg: "h-11 rounded-md px-8",
        icon: "h-10 w-10",
      },
    },
    /**
     * Phone-sized tap targets.
     *
     * The size scale is drawn for a pointer: `default` is 40px tall, `sm` is
     * 36 and `icon` is 40x40 — every one of them under the 44x44 WCAG 2.5.5
     * and the Apple HIG ask for, and they are the same buttons a thumb gets on
     * a phone. Growing them below `sm` fixes the whole set at once (the header
     * menu and theme toggles, "Back to jobs", "Add comment", the onboarding
     * Skip/Next pair) and leaves pointer layouts at exactly the density they
     * were designed with.
     *
     * `link` is deliberately left out: it renders as inline text inside a
     * sentence, where a 44px box would open a hole in the paragraph. Where a
     * link-styled control is a primary action it should be a real button.
     */
    compoundVariants: [
      {
        variant: ["default", "destructive", "outline", "secondary", "ghost"],
        size: ["default", "sm"],
        class: "max-sm:h-11",
      },
      {
        variant: ["default", "destructive", "outline", "secondary", "ghost"],
        size: "icon",
        class: "max-sm:h-11 max-sm:w-11",
      },
    ],
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);
