import { useState, useRef, useEffect, useCallback, type CSSProperties } from "react";
import { cn } from "@/shared/lib/utils";

interface SelectOption {
  readonly value: string;
  readonly label: string;
}

interface EditableSelectProps {
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly options: readonly SelectOption[];
  readonly placeholder?: string;
  readonly style?: CSSProperties;
  readonly className?: string;
  readonly editable?: boolean;
}

export function EditableSelect({
  value,
  onChange,
  options,
  placeholder = "Select...",
  style,
  className,
  editable = true,
}: EditableSelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  const selectedLabel = options.find((o) => o.value === value)?.label ?? "";

  const toggle = useCallback(() => {
    if (!editable) return;
    setIsOpen((prev) => !prev);
  }, [editable]);

  const handleSelect = useCallback(
    (optionValue: string) => {
      onChange(optionValue);
      setIsOpen(false);
    },
    [onChange],
  );

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;
    function handleClick(e: MouseEvent) {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [isOpen]);

  return (
    <div className="relative inline-block" style={style}>
      <span
        onClick={toggle}
        className={cn(
          className,
          editable && "cursor-pointer transition-colors rounded hover:bg-blue-50/50",
          !selectedLabel && editable && "italic text-gray-400",
        )}
      >
        {selectedLabel || placeholder}
      </span>

      {isOpen && (
        <div
          ref={popoverRef}
          className="absolute left-0 top-full z-50 mt-1 min-w-[120px] rounded-md border bg-white py-1 shadow-lg"
        >
          {options.map((option) => (
            <button
              key={option.value}
              onClick={() => handleSelect(option.value)}
              className={cn(
                "block w-full px-3 py-1 text-left text-xs hover:bg-blue-50 transition-colors",
                value === option.value && "bg-blue-50 font-medium",
              )}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
