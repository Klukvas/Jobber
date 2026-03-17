import { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Languages } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { cn } from "@/shared/lib/utils";

const languages = [
  { code: "en", label: "English" },
  { code: "ua", label: "Українська" },
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

  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  const iconClass = iconSize === "sm" ? "h-4 w-4" : "h-5 w-5";

  return (
    <div className="relative" ref={menuRef}>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setOpen(!open)}
        aria-label={t("common.changeLanguage")}
        aria-haspopup="menu"
        aria-expanded={open}
        className={cn(iconSize === "sm" ? "h-9 w-9" : undefined, className)}
      >
        <Languages className={iconClass} />
      </Button>
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div
            role="menu"
            className="absolute right-0 top-full z-50 mt-2 w-32 rounded-md border bg-popover p-1 shadow-md"
          >
            {languages.map(({ code, label }) => (
              <button
                key={code}
                role="menuitem"
                onClick={() => {
                  i18n.changeLanguage(code);
                  setOpen(false);
                }}
                className={cn(
                  "w-full rounded-sm px-3 py-2 text-left text-sm hover:bg-accent",
                  i18n.language === code && "bg-accent",
                )}
              >
                {label}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
