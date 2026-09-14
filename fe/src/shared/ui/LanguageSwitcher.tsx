import { useState, useEffect, useId, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Languages } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { cn } from "@/shared/lib/utils";
import { TAP_TARGET_ICON } from "@/shared/ui/tapTarget";

const languages = [
  { code: "en", label: "English" },
  { code: "uk", label: "Українська" },
  { code: "ru", label: "Русский" },
] as const;

interface LanguageSwitcherProps {
  readonly iconSize?: "sm" | "md";
  readonly className?: string;
}

export function LanguageSwitcher({
  iconSize = "md",
  className,
}: LanguageSwitcherProps) {
  const { t, i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const itemsRef = useRef<(HTMLButtonElement | null)[]>([]);
  // More than one of these is on the landing page at once, so the IDREF below
  // has to be unique per instance rather than a constant.
  const menuId = useId();

  /**
   * Closes the menu, putting the keyboard back where it came from.
   *
   * Escape used to close it and stop there. The element that had focus was the
   * menu item that had just been unmounted, and the browser drops focus off a
   * detached node onto `<body>` — so dismissing the menu left a keyboard user
   * at the top of the document with no way back to the control they opened.
   */
  const close = (returnFocus: boolean) => {
    setOpen(false);
    if (returnFocus) triggerRef.current?.focus();
  };

  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  // A menu takes the keyboard when it opens — otherwise the items announced by
  // `aria-haspopup="menu"` are somewhere the reader has to go looking for.
  // The current language is where it opens, which is what a radio group does.
  useEffect(() => {
    if (!open) return;
    const checked = languages.findIndex(({ code }) => code === i18n.language);
    itemsRef.current[checked === -1 ? 0 : checked]?.focus();
  }, [open, i18n.language]);

  const focusItem = (index: number) => {
    const last = languages.length - 1;
    const wrapped = index < 0 ? last : index > last ? 0 : index;
    itemsRef.current[wrapped]?.focus();
  };

  /**
   * Arrow keys move within the menu, Tab leaves it.
   *
   * That is what `role="menu"` promises: a screen reader in application mode
   * hands the arrows to the widget, and without this they scrolled the page
   * behind it instead of moving between the three languages.
   */
  const handleMenuKeyDown = (event: React.KeyboardEvent) => {
    const current = itemsRef.current.findIndex(
      (item) => item === document.activeElement,
    );

    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        focusItem(current + 1);
        break;
      case "ArrowUp":
        event.preventDefault();
        focusItem(current - 1);
        break;
      case "Home":
        event.preventDefault();
        focusItem(0);
        break;
      case "End":
        event.preventDefault();
        focusItem(languages.length - 1);
        break;
      case "Tab":
        // Leaving by Tab is leaving: the menu closes and the browser carries
        // focus on to whatever follows the trigger.
        close(false);
        break;
    }
  };

  const iconClass = iconSize === "sm" ? "h-4 w-4" : "h-5 w-5";

  return (
    <div className="relative" ref={menuRef}>
      <Button
        ref={triggerRef}
        variant="ghost"
        size="icon"
        onClick={() => setOpen(!open)}
        aria-label={t("common.changeLanguage")}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        // 44px wherever a finger is plausibly driving this. The small size is
        // the landing navbar's, which still shows its pointer layout at 768px
        // on a tablet, so it keeps the full target up to `lg` and absorbs the
        // difference into a negative margin rather than widening the cluster.
        className={cn(
          iconSize === "sm" ? TAP_TARGET_ICON : "h-11 w-11 sm:h-10 sm:w-10",
          className,
        )}
      >
        <Languages className={iconClass} />
      </Button>
      {open && (
        <div className="fixed inset-0 z-40" onClick={() => close(false)} />
      )}
      {/* Present in both states so the trigger's `aria-controls` always
          resolves — a name that points at nothing is the same to a screen
          reader as no name at all. `hidden` takes it out of the layout, the
          tab order and the accessibility tree while it is shut. */}
      <div
        id={menuId}
        role="menu"
        hidden={!open}
        onKeyDown={handleMenuKeyDown}
        className="absolute right-0 top-full z-50 mt-2 w-32 rounded-md border bg-popover p-1 shadow-md"
      >
        {open &&
          languages.map(({ code, label }, index) => (
            // menuitemradio + aria-checked: the highlight was purely visual,
            // so the current language was invisible to assistive tech.
            <button
              key={code}
              ref={(node) => {
                itemsRef.current[index] = node;
              }}
              role="menuitemradio"
              aria-checked={i18n.language === code}
              // One stop for the whole menu, as a menu has: the arrows move
              // inside it, Tab moves past it.
              tabIndex={i18n.language === code ? 0 : -1}
              onClick={() => {
                i18n.changeLanguage(code);
                close(true);
              }}
              // 44px per row on phones; the compact 32px pointer density
              // from `sm` up, matching the trigger above it. A three-item
              // menu at 44px is 132px tall on a desktop dropdown that has no
              // reason to be.
              className={cn(
                "flex min-h-11 w-full items-center rounded-sm px-3 py-2 text-left text-sm hover:bg-accent",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                "sm:min-h-8 sm:py-1.5",
                i18n.language === code && "bg-accent",
              )}
            >
              {label}
            </button>
          ))}
      </div>
    </div>
  );
}
