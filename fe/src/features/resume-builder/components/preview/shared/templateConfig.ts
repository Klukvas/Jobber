import type { ComponentType } from "react";

/**
 * How skills are rendered across templates.
 *
 * - "text-level":  name + (level) in flex-wrap  (professional, executive, minimal)
 * - "pill":        colored badge, white text, no level  (creative, elegant, iconic)
 * - "grid-level":  3-column grid with name + (level)  (compact)
 * - "vertical":    stacked name + level below  (modern)
 * - "text-only":   name-only dots join, no level  (minimal read-only variant)
 */
export type SkillRenderMode =
  | "text-level"
  | "pill"
  | "grid-level"
  | "vertical"
  | "text-only"
  | "dots"
  | "bar"
  | "square"
  | "star"
  | "circle"
  | "segmented"
  | "bubble";

/**
 * How languages are rendered across templates.
 *
 * - "flex":  flex-wrap with proficiency select  (most templates)
 * - "grid":  3-column grid with proficiency select  (compact)
 */
export type LanguageRenderMode = "flex" | "grid";

export type TemplateVariant =
  | "professional"
  | "modern"
  | "minimal"
  | "executive"
  | "creative"
  | "compact"
  | "elegant"
  | "iconic"
  | "bold"
  | "accent"
  | "timeline"
  | "vivid";

export interface TemplateConfig {
  /** SectionHeader variant. Also used as the template key. */
  readonly variant: TemplateVariant;

  /** Summary section heading text (e.g. "Professional Summary"). */
  readonly summaryTitle: string;

  /** Prefix prepended to SectionHeader titles (e.g. "diamond " for elegant). */
  readonly sectionTitlePrefix?: string;

  /** Base text size class applied to body copy. */
  readonly textSize: string;

  /** Leading class for body text. */
  readonly leadingClass: string;

  /** Skills rendering configuration. */
  readonly skills: {
    readonly renderAs: SkillRenderMode;
    readonly containerClass: string;
  };

  /** Languages rendering configuration. */
  readonly languages: {
    readonly renderAs: LanguageRenderMode;
    readonly containerClass: string;
  };

  /**
   * If present, each section key is wrapped with a colored-circle icon.
   * Maps section key -> lucide icon component.
   * Used only by the "iconic" template.
   */
  readonly sectionIcons?: Readonly<
    Record<string, ComponentType<{ className?: string }>>
  >;

  /**
   * When true, a "contact" case is expected in renderSection.
   * Used only by the "modern" template (renders contact in sidebar).
   */
  readonly renderContactInSwitch?: boolean;

  /**
   * Extra className applied to inline EditableField inputs.
   * Used for white text on colored backgrounds (modern sidebar).
   */
  readonly inputClassName?: string;

  /** Margin-bottom class for the summary section wrapper. */
  readonly summaryMb?: string;

  /** Per-section entry spacing overrides. */
  readonly entrySpacing?: {
    readonly experience?: string;
    readonly education?: string;
    readonly certification?: string;
    readonly project?: string;
    readonly volunteering?: string;
    readonly customSection?: string;
  };
}
