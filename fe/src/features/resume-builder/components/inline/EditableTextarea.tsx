import {
  useState,
  useRef,
  useCallback,
  useLayoutEffect,
  type CSSProperties,
  type KeyboardEvent,
} from "react";
import { cn } from "@/shared/lib/utils";
import { useInlineEditFocus } from "./useInlineEditFocus";

interface EditableTextareaProps {
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly placeholder?: string;
  readonly style?: CSSProperties;
  readonly className?: string;
  readonly editable?: boolean;
  readonly maxLength?: number;
  readonly "aria-describedby"?: string;
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
  maxLength,
  "aria-describedby": describedBy,
}: EditableTextareaProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const originalValue = useRef(value);
  const editing = useRef(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const startEditing = useCallback(() => {
    if (!editable) return;
    originalValue.current = value;
    editing.current = true;
    setDraft(value);
    setIsEditing(true);
  }, [editable, value]);

  useInlineEditFocus(textareaRef, isEditing);

  // Sized to its content as it opens, in the same commit as the focus above:
  // a textarea that grows a frame later moves the text under the caret.
  useLayoutEffect(() => {
    if (!isEditing) return;
    const el = textareaRef.current;
    if (el) autoResize(el);
  }, [isEditing]);

  const commit = useCallback(() => {
    if (!editing.current) return;
    editing.current = false;
    setIsEditing(false);
    if (draft !== value) {
      onChange(draft);
    }
  }, [draft, value, onChange]);

  const cancel = useCallback(() => {
    editing.current = false;
    setIsEditing(false);
    setDraft(originalValue.current);
    if (draft !== originalValue.current) onChange(originalValue.current);
  }, [draft, onChange]);

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
      const next = e.target.value;
      // An existing oversized value remains editable downward, never silently cut.
      if (maxLength !== undefined && next.length > maxLength && next.length >= draft.length) return;
      setDraft(next);
      onChange(next);
      autoResize(e.target);
    },
    [draft.length, maxLength, onChange],
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
        aria-describedby={describedBy}
        className={cn(
          "w-full resize-none border-none bg-blue-50/70 outline-none ring-1 ring-blue-300 rounded px-0.5 -mx-0.5",
          className,
        )}
        style={style}
        rows={2}
        maxLength={maxLength}
      />
    );
  }

  const isEmpty = !value.trim();
  if (!editable && isEmpty) return null;

  return (
    <p
      onMouseDown={handleMouseDown}
      onKeyDown={editable ? handleDisplayKeyDown : undefined}
      tabIndex={editable ? 0 : undefined}
      role={editable ? "textbox" : undefined}
      aria-label={editable ? placeholder : undefined}
      aria-describedby={editable ? describedBy : undefined}
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
