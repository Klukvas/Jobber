import type { TemplateConfig } from "../templateConfig";
import type { TemplateSetup } from "../useTemplateSetup";

/** Shared props for all section content components. */
export interface SectionContentProps {
  readonly setup: TemplateSetup;
  readonly config: TemplateConfig;
  readonly editable: boolean;
  readonly sectionColor?: string;
  readonly inputClassName?: string;
}
