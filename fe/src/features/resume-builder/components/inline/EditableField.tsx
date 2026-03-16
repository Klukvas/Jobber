import {
  useState,
  useRef,
  useCallback,
  useEffect,
  type CSSProperties,
  type KeyboardEvent,
} from "react";
import { cn } from "@/shared/lib/utils";

type TagType = "span" | "p" | "h1" | "h2" | "h3";
type InputType = "text" | "date" | "email" | "url" | "tel";

interface EditableFieldProps {
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly placeholder?: string;
  readonly as?: TagType;
  readonly type?: InputType;
  readonly style?: CSSProperties;
  readonly className?: string;
  readonly inputClassName?: string;
  readonly editable?: boolean;
}

export function EditableField({
  value,
  onChange,
  placeholder = "",
  as: Tag = "span",
  type = "text",
  style,
  className,
  inputClassName,
  editable = true,
}: EditableFieldProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);

  const startEditing = useCallback(() => {
    if (!editable) return;
    setDraft(value);
    setIsEditing(true);
  }, [editable, value]);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      const el = inputRef.current;
      requestAnimationFrame(() => el.focus());
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

  const handleInputKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        e.preventDefault();
        commit();
      } else if (e.key === "Escape") {
        e.preventDefault();
        cancel();
      }
    },
    [commit, cancel],
  );

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!editable) return;
      e.preventDefault();
      startEditing();
    },
    [editable, startEditing],
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

  if (isEditing) {
    return (
      <input
        ref={inputRef}
        type={type}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={handleInputKeyDown}
        placeholder={placeholder}
        aria-label={placeholder}
        className={cn(
          "w-full border-none bg-blue-50/70 outline-none ring-1 ring-blue-300 rounded px-0.5 -mx-0.5",
          inputClassName ?? className,
        )}
        style={style}
      />
    );
  }

  const isEmpty = !value.trim();

  return (
    <Tag
      onMouseDown={handleMouseDown}
      onKeyDown={editable ? handleDisplayKeyDown : undefined}
      tabIndex={editable ? 0 : undefined}
      role={editable ? "textbox" : undefined}
      aria-label={editable ? placeholder : undefined}
      className={cn(
        className,
        editable &&
          "cursor-text transition-colors rounded hover:bg-blue-50/50 focus:outline-none focus:ring-1 focus:ring-blue-300",
        isEmpty && editable && "italic text-gray-400",
      )}
      style={style}
    >
      {isEmpty ? placeholder : value}
    </Tag>
  );
}
