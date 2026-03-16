import {
  useState,
  useRef,
  useCallback,
  useEffect,
  type CSSProperties,
  type KeyboardEvent,
} from "react";
import { cn } from "@/shared/lib/utils";

interface EditableTextareaProps {
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly placeholder?: string;
  readonly style?: CSSProperties;
  readonly className?: string;
  readonly editable?: boolean;
}

function autoResize(el: HTMLTextAreaElement) {
  el.style.height = "auto";
  el.style.height = `${el.scrollHeight}px`;
}

export function EditableTextarea({
  value,
  onChange,
  placeholder = "",
  style,
  className,
  editable = true,
}: EditableTextareaProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const startEditing = useCallback(() => {
    if (!editable) return;
    setDraft(value);
    setIsEditing(true);
  }, [editable, value]);

  useEffect(() => {
    if (isEditing && textareaRef.current) {
      const el = textareaRef.current;
      // Use rAF to ensure the textarea is fully mounted and painted
      // before focusing — prevents the "click to select, click again to type" issue
      requestAnimationFrame(() => {
        el.focus();
        autoResize(el);
      });
    }
  }, [isEditing]);

  const commit = useCallback(() => {
    setIsEditing(false);
    if (draft !== value) {
      onChange(draft);
    }
  }, [draft, value, onChange]);

  const cancel = useCallback(() => {
    setIsEditing(false);
    setDraft(value);
  }, [value]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Escape") {
        e.preventDefault();
        cancel();
      }
    },
    [cancel],
  );

  const handleDisplayKeyDown = useCallback(
    (e: KeyboardEvent<HTMLElement>) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        startEditing();
      }
    },
    [startEditing],
  );

  // Prevent mousedown from starting text selection on the <p>.
  // Without this, the browser selection fights with the textarea focus
  // after the component swaps, especially inside scaled containers.
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!editable) return;
      e.preventDefault();
      startEditing();
    },
    [editable, startEditing],
  );

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setDraft(e.target.value);
      autoResize(e.target);
    },
    [],
  );

  if (isEditing) {
    return (
      <textarea
        ref={textareaRef}
        value={draft}
        onChange={handleChange}
        onBlur={commit}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        aria-label={placeholder}
        className={cn(
          "w-full resize-none border-none bg-blue-50/70 outline-none ring-1 ring-blue-300 rounded px-0.5 -mx-0.5",
          className,
        )}
        style={style}
        rows={2}
      />
    );
  }

  const isEmpty = !value.trim();

  return (
    <p
      onMouseDown={handleMouseDown}
      onKeyDown={editable ? handleDisplayKeyDown : undefined}
      tabIndex={editable ? 0 : undefined}
      role={editable ? "textbox" : undefined}
      aria-label={editable ? placeholder : undefined}
      className={cn(
        "whitespace-pre-line",
        className,
        editable &&
          "cursor-text transition-colors rounded hover:bg-blue-50/50 focus:outline-none focus:ring-1 focus:ring-blue-300",
        isEmpty && editable && "italic text-gray-400",
      )}
      style={style}
    >
      {isEmpty ? placeholder : value}
    </p>
  );
}
